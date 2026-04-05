# MQTT Skill Guide

## Overview

MQTT (Message Queuing Telemetry Transport) is a lightweight pub/sub protocol designed for constrained devices and unreliable networks. Use for IoT telemetry, device command/control, and low-bandwidth messaging.

## Broker Setup

### Mosquitto (Self-Hosted)

```bash
# Install
apt-get install mosquitto mosquitto-clients

# mosquitto.conf
listener 1883                    # Plain TCP
listener 8883                    # TLS
cafile /etc/ssl/ca.crt
certfile /etc/ssl/server.crt
keyfile /etc/ssl/server.key

listener 9001                    # WebSocket for browsers
protocol websockets

allow_anonymous false
password_file /etc/mosquitto/passwd
# persistence
persistence true
persistence_location /var/lib/mosquitto/

# Logging
log_type all
log_dest file /var/log/mosquitto/mosquitto.log
```

```bash
# Create password file
mosquitto_passwd -c /etc/mosquitto/passwd myuser

# Test publish/subscribe
mosquitto_pub -h localhost -t sensors/device1/temperature -m '{"value":22.5}'
mosquitto_sub -h localhost -t sensors/+/temperature
```

### EMQX (Production-Grade)

```yaml
# docker-compose.yml
services:
  emqx:
    image: emqx/emqx:5.4.0
    ports:
      - "1883:1883"    # MQTT
      - "8883:8883"    # MQTT/TLS
      - "8083:8083"    # MQTT/WebSocket
      - "8084:8084"    # MQTT/WSS
      - "18083:18083"  # Dashboard
    environment:
      EMQX_NAME: emqx
      EMQX_DASHBOARD__DEFAULT_PASSWORD: changeme
    volumes:
      - emqx_data:/opt/emqx/data
```

## QoS Levels

| Level | Name | Guarantee | Use For |
|-------|------|-----------|---------|
| QoS 0 | At most once | Fire and forget — may be lost | High-frequency sensor data where occasional loss is acceptable |
| QoS 1 | At least once | Guaranteed delivery, may duplicate | Commands, alerts where duplicates are idempotent |
| QoS 2 | Exactly once | Guaranteed, no duplicates (4-way handshake) | Billing events, critical state changes |

```python
# paho-mqtt Python
client.publish("sensors/device1/temp", payload='{"value":22.5}', qos=0)  # fire-and-forget
client.publish("devices/device1/cmd", payload='{"action":"reboot"}', qos=1)  # at-least-once
client.publish("billing/events", payload='{"amount":99.99}', qos=2)  # exactly-once
```

## Topic Hierarchy Design

Use `/` as delimiter to create a logical hierarchy. Wildcards: `+` (single level), `#` (multi-level).

```
# Structure: domain/entityId/measurement
sensors/{deviceId}/temperature
sensors/{deviceId}/humidity
sensors/{deviceId}/battery

devices/{deviceId}/commands        # server → device
devices/{deviceId}/status          # device → server (current state)
devices/{deviceId}/ota             # firmware updates

fleet/{fleetId}/vehicles/{vehicleId}/location
users/{userId}/notifications
rooms/{roomId}/messages
```

```bash
# Wildcard subscriptions
sensors/+/temperature         # all device temperatures (single level)
sensors/#                     # all sensor data for all devices (multi-level)
fleet/fleet1/#                # all data for fleet1
```

## Retained Messages

A retained message is stored by the broker and immediately sent to new subscribers. Use for "last known state".

```python
import paho.mqtt.client as mqtt

client = mqtt.Client(client_id="server")
client.connect("localhost", 1883)

# Publish retained — new subscribers get this immediately
client.publish(
    topic="devices/device1/status",
    payload='{"online":true,"firmware":"1.2.3"}',
    qos=1,
    retain=True,   # broker stores this as the topic's last value
)

# Clear retained message by publishing empty payload
client.publish("devices/device1/status", payload="", retain=True)
```

## Last Will and Testament (LWT)

LWT is a message the broker publishes automatically when a client disconnects unexpectedly (without sending DISCONNECT).

```python
import paho.mqtt.client as mqtt

client = mqtt.Client(client_id="device-abc")

# Set LWT before connecting
client.will_set(
    topic="devices/device-abc/status",
    payload='{"online":false,"reason":"unexpected_disconnect"}',
    qos=1,
    retain=True,
)

client.connect("localhost", 1883, keepalive=60)
client.loop_start()

# On clean shutdown, publish online=false yourself before disconnect
client.publish("devices/device-abc/status", '{"online":false,"reason":"shutdown"}', qos=1, retain=True)
client.disconnect()
```

## Client Library Examples

### Python (paho-mqtt)

```python
import json
import paho.mqtt.client as mqtt

def on_connect(client, userdata, flags, rc):
    if rc == 0:
        print("Connected to broker")
        # Subscribe after connect (resubscribes on reconnect)
        client.subscribe([
            ("sensors/+/temperature", 0),
            ("devices/+/status", 1),
        ])
    else:
        print(f"Connection failed: {rc}")

def on_message(client, userdata, msg):
    try:
        payload = json.loads(msg.payload.decode())
        print(f"Topic: {msg.topic}, QoS: {msg.qos}, Payload: {payload}")
    except json.JSONDecodeError as e:
        print(f"Invalid JSON on {msg.topic}: {e}")

def on_disconnect(client, userdata, rc):
    if rc != 0:
        print(f"Unexpected disconnect: {rc}, will reconnect")

client = mqtt.Client(client_id="backend-service", clean_session=True)
client.username_pw_set("myuser", "mypass")
client.on_connect = on_connect
client.on_message = on_message
client.on_disconnect = on_disconnect

# TLS
client.tls_set(ca_certs="/etc/ssl/ca.crt")

# Auto-reconnect
client.connect("localhost", 8883, keepalive=60)
client.reconnect_delay_set(min_delay=1, max_delay=30)
client.loop_forever()  # blocking; use loop_start() for async
```

### Node.js (mqtt.js)

```typescript
import mqtt, { MqttClient } from "mqtt";

const client: MqttClient = mqtt.connect("mqtt://localhost:1883", {
  clientId: "backend-service",
  username: "myuser",
  password: "mypass",
  clean: true,
  reconnectPeriod: 5000,   // ms between reconnects
  connectTimeout: 30000,
  will: {
    topic: "services/backend/status",
    payload: JSON.stringify({ online: false }),
    qos: 1,
    retain: true,
  },
});

client.on("connect", () => {
  console.log("Connected to MQTT broker");
  client.subscribe(["sensors/+/temperature", "devices/+/cmd"], { qos: 1 }, (err) => {
    if (err) console.error("Subscribe error:", err);
  });
  // Announce online
  client.publish("services/backend/status", JSON.stringify({ online: true }), { qos: 1, retain: true });
});

client.on("message", (topic: string, payload: Buffer) => {
  try {
    const data = JSON.parse(payload.toString());
    routeMessage(topic, data);
  } catch (err) {
    console.error(`Invalid message on ${topic}:`, err);
  }
});

client.on("error", (err) => console.error("MQTT error:", err));
client.on("reconnect", () => console.log("Reconnecting to MQTT broker..."));

function routeMessage(topic: string, data: unknown) {
  const parts = topic.split("/");
  if (parts[0] === "sensors" && parts[2] === "temperature") {
    handleTemperature(parts[1], data);
  }
}

function publish(topic: string, payload: unknown, qos: 0 | 1 | 2 = 1) {
  client.publish(topic, JSON.stringify(payload), { qos });
}
```

## MQTT over WebSocket (Browser)

```typescript
// Browsers can't use raw TCP MQTT — use WebSocket transport
import mqtt from "mqtt";

const client = mqtt.connect("wss://broker.example.com:8084/mqtt", {
  clientId: `browser-${Math.random().toString(16).slice(2)}`,
  username: "webuser",
  password: "webpass",
  clean: true,
  protocolVersion: 5,   // MQTT v5 for enhanced features
});

client.on("connect", () => {
  client.subscribe("notifications/user123/#", { qos: 1 });
});

client.on("message", (topic, payload) => {
  const event = JSON.parse(payload.toString());
  displayNotification(event);
});
```

## Rules

- Always subscribe inside `on_connect` callback — subscriptions are lost on reconnect unless re-established
- Use `retain=True` for state topics (device online/offline, current config) so new subscribers get current state
- Set LWT before connecting to ensure offline detection even on network failure
- Use `clean_session=False` (persistent sessions) with QoS 1/2 for devices that need missed messages on reconnect
- Never use `#` wildcard subscription for high-throughput topics — it processes every message on the broker
- Topic names are case-sensitive and cannot start with `$` (reserved for broker system topics like `$SYS/#`)
- Validate and sanitize topic segments from user input — topic injection can expose other clients' data
