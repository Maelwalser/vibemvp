# WebSockets Skill Guide

## Overview

WebSockets provide full-duplex communication over a single persistent TCP connection. Use for real-time features: chat, live dashboards, collaborative editing, notifications.

## Connection State Management

Track each connected client with a Map keyed by a unique client ID.

### Node.js (ws library)

```typescript
import { WebSocketServer, WebSocket } from "ws";
import { IncomingMessage } from "http";
import { v4 as uuidv4 } from "uuid";

interface Client {
  id: string;
  ws: WebSocket;
  userId?: string;
  roomId?: string;
}

// Global connection registry
const clients = new Map<string, Client>();

const wss = new WebSocketServer({ port: 8080 });

wss.on("connection", (ws: WebSocket, req: IncomingMessage) => {
  const clientId = uuidv4();
  const client: Client = { id: clientId, ws };
  clients.set(clientId, client);

  console.log(`Client connected: ${clientId}. Total: ${clients.size}`);

  ws.on("message", (data) => handleMessage(clientId, data));

  ws.on("close", () => {
    clients.delete(clientId);
    console.log(`Client disconnected: ${clientId}. Total: ${clients.size}`);
  });

  ws.on("error", (err) => {
    console.error(`Client error ${clientId}:`, err);
    clients.delete(clientId);
  });

  // Send welcome message
  send(ws, { type: "connected", clientId });
});
```

### Go (gorilla/websocket)

```go
package ws

import (
    "sync"
    "github.com/google/uuid"
    "github.com/gorilla/websocket"
)

type Client struct {
    ID     string
    Conn   *websocket.Conn
    UserID string
    RoomID string
    send   chan []byte
}

type Hub struct {
    clients map[string]*Client
    mu      sync.RWMutex
}

func NewHub() *Hub {
    return &Hub{clients: make(map[string]*Client)}
}

func (h *Hub) Register(conn *websocket.Conn) *Client {
    client := &Client{
        ID:   uuid.NewString(),
        Conn: conn,
        send: make(chan []byte, 256),
    }
    h.mu.Lock()
    h.clients[client.ID] = client
    h.mu.Unlock()
    return client
}

func (h *Hub) Unregister(id string) {
    h.mu.Lock()
    delete(h.clients, id)
    h.mu.Unlock()
}
```

## Message Types with Discriminator

Always use a `type` field so clients and servers can route messages correctly.

```typescript
// Message type definitions
type Message =
  | { type: "chat"; roomId: string; content: string; senderId: string }
  | { type: "join_room"; roomId: string }
  | { type: "leave_room"; roomId: string }
  | { type: "ping" }
  | { type: "pong" }
  | { type: "error"; code: string; message: string }
  | { type: "connected"; clientId: string };

function send(ws: WebSocket, msg: Message) {
  if (ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(msg));
  }
}

function handleMessage(clientId: string, data: Buffer | string) {
  let msg: Message;
  try {
    msg = JSON.parse(data.toString());
  } catch {
    const client = clients.get(clientId);
    if (client) send(client.ws, { type: "error", code: "invalid_json", message: "Invalid JSON" });
    return;
  }

  const client = clients.get(clientId);
  if (!client) return;

  switch (msg.type) {
    case "join_room":
      client.roomId = msg.roomId;
      break;
    case "leave_room":
      delete client.roomId;
      break;
    case "chat":
      broadcastToRoom(msg.roomId, msg, clientId);
      break;
    case "ping":
      send(client.ws, { type: "pong" });
      break;
  }
}
```

## Broadcast Patterns

### Broadcast to All Connected Clients

```typescript
function broadcastToAll(msg: Message, excludeClientId?: string) {
  const payload = JSON.stringify(msg);
  for (const [id, client] of clients) {
    if (id !== excludeClientId && client.ws.readyState === WebSocket.OPEN) {
      client.ws.send(payload);
    }
  }
}
```

### Broadcast to Room

```typescript
function broadcastToRoom(roomId: string, msg: Message, excludeClientId?: string) {
  const payload = JSON.stringify(msg);
  for (const [id, client] of clients) {
    if (
      client.roomId === roomId &&
      id !== excludeClientId &&
      client.ws.readyState === WebSocket.OPEN
    ) {
      client.ws.send(payload);
    }
  }
}
```

### Go Broadcast

```go
func (h *Hub) BroadcastToRoom(roomID string, payload []byte, excludeID string) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    for id, client := range h.clients {
        if id != excludeID && client.RoomID == roomID {
            select {
            case client.send <- payload:
            default:
                // Slow client — drop or disconnect
                close(client.send)
                delete(h.clients, id)
            }
        }
    }
}
```

## Heartbeat Ping/Pong

Detect dead connections with periodic pings. If a client doesn't respond within the deadline, close the connection.

```typescript
const PING_INTERVAL_MS = 30_000;
const PONG_TIMEOUT_MS = 10_000;

function setupHeartbeat(clientId: string) {
  const client = clients.get(clientId);
  if (!client) return;

  let pongReceived = true;

  const interval = setInterval(() => {
    if (!pongReceived) {
      console.log(`Client ${clientId} failed heartbeat — closing`);
      client.ws.terminate();
      clients.delete(clientId);
      clearInterval(interval);
      return;
    }
    pongReceived = false;
    if (client.ws.readyState === WebSocket.OPEN) {
      client.ws.ping(); // ws library sends WebSocket protocol ping frame
    }
  }, PING_INTERVAL_MS);

  client.ws.on("pong", () => {
    pongReceived = true;
  });

  client.ws.on("close", () => clearInterval(interval));
}
```

### Go Heartbeat

```go
func (c *Client) writePump() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case msg, ok := <-c.send:
            c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if !ok {
                c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }
            c.Conn.WriteMessage(websocket.TextMessage, msg)

        case <-ticker.C:
            c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}

func (c *Client) readPump() {
    c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    c.Conn.SetPongHandler(func(string) error {
        c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })
    // ...
}
```

## Reconnection Handling (Client Side)

```typescript
class ReconnectingWebSocket {
  private ws: WebSocket | null = null;
  private reconnectDelay = 1000;
  private maxReconnectDelay = 30000;
  private reconnectAttempts = 0;

  constructor(private url: string) {
    this.connect();
  }

  private connect() {
    this.ws = new WebSocket(this.url);

    this.ws.onopen = () => {
      console.log("Connected");
      this.reconnectDelay = 1000; // reset backoff
      this.reconnectAttempts = 0;
    };

    this.ws.onmessage = (event) => {
      this.onMessage(JSON.parse(event.data));
    };

    this.ws.onclose = (event) => {
      if (!event.wasClean) {
        this.scheduleReconnect();
      }
    };

    this.ws.onerror = () => {
      this.ws?.close();
    };
  }

  private scheduleReconnect() {
    this.reconnectAttempts++;
    const jitter = Math.random() * 1000;
    const delay = Math.min(this.reconnectDelay * 2 ** this.reconnectAttempts, this.maxReconnectDelay) + jitter;
    console.log(`Reconnecting in ${Math.round(delay)}ms (attempt ${this.reconnectAttempts})`);
    setTimeout(() => this.connect(), delay);
  }

  send(msg: object) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(msg));
    }
  }

  onMessage(_msg: unknown) {} // override in subclass
}
```

## HTTP Upgrade (Go net/http)

```go
var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        // Validate origin — NEVER return true unconditionally in production
        origin := r.Header.Get("Origin")
        return origin == "https://app.example.com"
    },
}

func wsHandler(hub *Hub) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        conn, err := upgrader.Upgrade(w, r, nil)
        if err != nil {
            http.Error(w, "upgrade failed", http.StatusBadRequest)
            return
        }
        client := hub.Register(conn)
        go client.writePump()
        go client.readPump()
    }
}
```

## Rules

- Always check `ws.readyState === WebSocket.OPEN` before sending
- Validate the `Origin` header in the upgrade handshake to prevent CSRF
- Use a goroutine-per-client with a buffered send channel (Go) or async event handlers (Node)
- Never block the message handler — offload slow work to async tasks or goroutines
- Implement heartbeat to detect zombie connections (no TCP FIN on mobile network drops)
- Use exponential backoff with jitter for reconnection — never reconnect immediately in a tight loop
- Authenticate at connection time via query param token or cookie, not per-message
