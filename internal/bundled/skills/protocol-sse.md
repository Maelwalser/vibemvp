# Server-Sent Events (SSE) Skill Guide

## Overview

Server-Sent Events (SSE) is a unidirectional HTTP-based push mechanism: the server streams events to the browser over a single long-lived connection. Use for live feeds, notifications, progress updates, and log streaming. Choose WebSockets when bidirectional communication is needed.

## Event Stream Format

SSE uses `text/event-stream` content type. Each event is a block of newline-separated fields terminated by a blank line.

```
# Comment line (ignored by client)

data: Simple string message\n\n

event: user-joined\n
data: {"userId":"abc123","name":"Alice"}\n\n

id: 42\n
event: message\n
data: {"text":"Hello"}\n
retry: 3000\n\n
```

Field reference:

| Field | Purpose |
|-------|---------|
| `data:` | Event payload (required). Use multiple `data:` lines for multi-line content |
| `event:` | Named event type. Client uses `addEventListener(name)`. Default: `message` |
| `id:` | Event ID. Browser sends as `Last-Event-ID` header on reconnect |
| `retry:` | Reconnect delay in ms. Overrides browser default (~3s) |

## EventSource Client

```typescript
const es = new EventSource("/api/events", { withCredentials: true });

// Default "message" event
es.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log("Received:", data);
};

// Named event listeners
es.addEventListener("user-joined", (event) => {
  const { userId, name } = JSON.parse(event.data);
  console.log(`${name} joined`);
});

es.addEventListener("notification", (event) => {
  showNotification(JSON.parse(event.data));
});

// Connection state
es.onopen = () => console.log("SSE connected, readyState:", es.readyState);
es.onerror = (event) => {
  if (es.readyState === EventSource.CLOSED) {
    console.log("SSE connection closed");
  } else {
    console.log("SSE error — browser will reconnect automatically");
  }
};

// Close when done
// es.close();
```

### Last-Event-ID for Resumability

```typescript
// Browser automatically sends Last-Event-ID header on reconnect
// Server uses it to replay missed events

// Server checks req.headers['last-event-id'] and streams from that ID forward
```

## Implementations Per Framework

### Express (Node.js)

```typescript
import express, { Request, Response } from "express";

const app = express();

// SSE helper
function sseMiddleware(req: Request, res: Response, next: Function) {
  res.setHeader("Content-Type", "text/event-stream");
  res.setHeader("Cache-Control", "no-cache");
  res.setHeader("Connection", "keep-alive");
  res.setHeader("X-Accel-Buffering", "no"); // disable nginx buffering
  res.flushHeaders();
  next();
}

// Write helpers
function writeEvent(res: Response, opts: {
  data: unknown;
  event?: string;
  id?: string | number;
  retry?: number;
}) {
  if (opts.retry) res.write(`retry: ${opts.retry}\n`);
  if (opts.id !== undefined) res.write(`id: ${opts.id}\n`);
  if (opts.event) res.write(`event: ${opts.event}\n`);
  res.write(`data: ${JSON.stringify(opts.data)}\n\n`);
}

// SSE endpoint
app.get("/api/events", sseMiddleware, (req: Request, res: Response) => {
  const lastEventId = req.headers["last-event-id"];
  let counter = lastEventId ? parseInt(lastEventId as string) + 1 : 0;

  // Send missed events if resuming
  if (lastEventId) {
    const missed = eventStore.getFrom(parseInt(lastEventId as string));
    for (const event of missed) {
      writeEvent(res, event);
    }
  }

  // Stream live events
  const interval = setInterval(() => {
    writeEvent(res, {
      id: counter++,
      event: "heartbeat",
      data: { timestamp: new Date().toISOString() },
    });
  }, 15000);

  // Subscribe to real events
  const unsubscribe = eventBus.subscribe((event) => {
    writeEvent(res, { id: counter++, event: event.type, data: event.payload });
  });

  req.on("close", () => {
    clearInterval(interval);
    unsubscribe();
  });
});
```

### FastAPI (Python)

```python
from fastapi import FastAPI, Request
from fastapi.responses import StreamingResponse
import asyncio
import json

app = FastAPI()

async def event_generator(request: Request, last_event_id: str | None):
    counter = int(last_event_id) + 1 if last_event_id else 0

    async def stream():
        nonlocal counter
        while True:
            if await request.is_disconnected():
                break

            event_data = {"id": counter, "timestamp": "..."}
            yield f"id: {counter}\n"
            yield f"event: update\n"
            yield f"data: {json.dumps(event_data)}\n\n"
            counter += 1

            await asyncio.sleep(1)

    return stream()

@app.get("/api/events")
async def sse_endpoint(request: Request):
    last_event_id = request.headers.get("last-event-id")
    generator = await event_generator(request, last_event_id)
    return StreamingResponse(
        generator,
        media_type="text/event-stream",
        headers={
            "Cache-Control": "no-cache",
            "X-Accel-Buffering": "no",
        },
    )
```

### Go (net/http)

```go
package handler

import (
    "fmt"
    "net/http"
    "strconv"
    "time"
)

func SSEHandler(eventBus EventBus) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // SSE headers
        w.Header().Set("Content-Type", "text/event-stream")
        w.Header().Set("Cache-Control", "no-cache")
        w.Header().Set("Connection", "keep-alive")
        w.Header().Set("X-Accel-Buffering", "no")

        flusher, ok := w.(http.Flusher)
        if !ok {
            http.Error(w, "streaming not supported", http.StatusInternalServerError)
            return
        }

        // Resume from last event
        var counter int64
        if lastID := r.Header.Get("Last-Event-ID"); lastID != "" {
            n, _ := strconv.ParseInt(lastID, 10, 64)
            counter = n + 1
        }

        // Send heartbeat and events
        ticker := time.NewTicker(15 * time.Second)
        defer ticker.Stop()

        events := eventBus.Subscribe()
        defer eventBus.Unsubscribe(events)

        for {
            select {
            case <-r.Context().Done():
                return

            case <-ticker.C:
                fmt.Fprintf(w, "id: %d\nevent: heartbeat\ndata: {}\n\n", counter)
                counter++
                flusher.Flush()

            case evt := <-events:
                fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", counter, evt.Type, evt.JSON())
                counter++
                flusher.Flush()
            }
        }
    }
}
```

## Multi-Client Fan-Out (Node.js)

```typescript
// Manage active SSE connections
const connections = new Set<Response>();

function fanOut(event: string, data: unknown) {
  const payload = `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`;
  for (const res of connections) {
    res.write(payload);
  }
}

app.get("/api/events", sseMiddleware, (req: Request, res: Response) => {
  connections.add(res);
  req.on("close", () => connections.delete(res));
});

// Emit from anywhere in the app
eventBus.on("order.created", (order) => fanOut("order-created", order));
```

## Rules

- Always set `Cache-Control: no-cache` and `Connection: keep-alive` headers
- Set `X-Accel-Buffering: no` to prevent nginx from buffering the stream
- Send keep-alive comments (`:\n\n`) or heartbeat events every 15–30s to prevent proxy/LB timeouts
- Use event `id:` fields to enable resumability via `Last-Event-ID`
- Clean up subscriptions and timers when the request closes (client disconnects)
- SSE uses HTTP/1.1 persistent connection — browsers cap at 6 connections per origin per HTTP/1.1, so use HTTP/2 or limit SSE tabs
- Use SSE for server-to-client push; use WebSockets if the client also needs to send data at high frequency
