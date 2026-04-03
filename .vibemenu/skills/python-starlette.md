# Python + Starlette Skill Guide

## Project Layout

```
service-name/
├── pyproject.toml
├── main.py              # App entry point
├── app/
│   ├── __init__.py
│   ├── routes.py        # Route() and Mount() definitions
│   ├── middleware.py    # Custom Middleware classes
│   ├── endpoints/       # Endpoint classes
│   ├── websockets/      # WebSocket endpoint classes
│   └── config.py
└── tests/
```

## Dependencies

```toml
[project]
dependencies = [
    "starlette>=0.37.0",
    "uvicorn[standard]>=0.29.0",
    "httpx>=0.27.0",   # for TestClient
]
```

## App Setup with Lifespan

```python
# main.py
from contextlib import asynccontextmanager
from starlette.applications import Starlette
from starlette.middleware import Middleware
from starlette.middleware.cors import CORSMiddleware
from starlette.routing import Route, Mount
from app.endpoints.users import UserListEndpoint, UserDetailEndpoint
from app.websockets.chat import ChatEndpoint
from app.database import engine

@asynccontextmanager
async def lifespan(app):
    # Startup: open DB pool
    await engine.connect()
    yield
    # Shutdown: close DB pool
    await engine.dispose()

middleware = [
    Middleware(CORSMiddleware, allow_origins=["*"], allow_methods=["*"]),
]

routes = [
    Route("/health", endpoint=health),
    Mount("/users", routes=[
        Route("/", endpoint=UserListEndpoint),
        Route("/{user_id:int}", endpoint=UserDetailEndpoint),
    ]),
    Route("/ws/chat", endpoint=ChatEndpoint),
]

app = Starlette(routes=routes, middleware=middleware, lifespan=lifespan)

async def health(request):
    from starlette.responses import JSONResponse
    return JSONResponse({"status": "ok"})
```

## Endpoint Classes

```python
# app/endpoints/users.py
from starlette.endpoints import HTTPEndpoint
from starlette.requests import Request
from starlette.responses import JSONResponse
from starlette.exceptions import HTTPException

class UserListEndpoint(HTTPEndpoint):
    async def get(self, request: Request) -> JSONResponse:
        users = await request.state.db.fetch_all("SELECT * FROM users")
        return JSONResponse([dict(u) for u in users])

    async def post(self, request: Request) -> JSONResponse:
        data = await request.json()
        if not data.get("email"):
            raise HTTPException(status_code=400, detail="email is required")
        user = await request.state.db.fetch_one(
            "INSERT INTO users (email) VALUES (:email) RETURNING *",
            {"email": data["email"]},
        )
        return JSONResponse(dict(user), status_code=201)

class UserDetailEndpoint(HTTPEndpoint):
    async def get(self, request: Request) -> JSONResponse:
        user_id = request.path_params["user_id"]
        user = await request.state.db.fetch_one(
            "SELECT * FROM users WHERE id = :id", {"id": user_id}
        )
        if user is None:
            raise HTTPException(status_code=404, detail="User not found")
        return JSONResponse(dict(user))
```

## WebSocket Endpoint

```python
# app/websockets/chat.py
from starlette.endpoints import WebSocketEndpoint
from starlette.websockets import WebSocket

class ChatEndpoint(WebSocketEndpoint):
    encoding = "text"

    async def on_connect(self, websocket: WebSocket) -> None:
        await websocket.accept()

    async def on_receive(self, websocket: WebSocket, data: str) -> None:
        await websocket.send_text(f"echo: {data}")

    async def on_disconnect(self, websocket: WebSocket, close_code: int) -> None:
        pass  # cleanup if needed
```

## Background Tasks

```python
from starlette.background import BackgroundTasks
from starlette.responses import JSONResponse

async def send_welcome_email(email: str) -> None:
    # I/O-bound fire-and-forget
    pass

async def register(request: Request) -> JSONResponse:
    data = await request.json()
    tasks = BackgroundTasks()
    tasks.add_task(send_welcome_email, data["email"])
    return JSONResponse({"status": "registered"}, background=tasks)
```

## Streaming Response

```python
from starlette.responses import StreamingResponse
import asyncio

async def event_generator():
    for i in range(10):
        yield f"data: {i}\n\n"
        await asyncio.sleep(1)

async def stream_events(request: Request) -> StreamingResponse:
    return StreamingResponse(
        event_generator(),
        media_type="text/event-stream",
        headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"},
    )
```

## Custom Middleware

```python
# app/middleware.py
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request
from starlette.responses import Response
import time

class TimingMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next) -> Response:
        start = time.perf_counter()
        response = await call_next(request)
        elapsed = time.perf_counter() - start
        response.headers["X-Process-Time"] = str(elapsed)
        return response
```

## Error Handling

- Raise `HTTPException(status_code=..., detail="...")` inside endpoints.
- Add `exception_handlers` to `Starlette(...)` for global error shapes:

```python
from starlette.responses import JSONResponse

async def http_exception_handler(request, exc):
    return JSONResponse({"error": exc.detail}, status_code=exc.status_code)

app = Starlette(
    routes=routes,
    exception_handlers={HTTPException: http_exception_handler},
)
```

## Key Rules

- Use `lifespan` async context manager — `@app.on_event` is deprecated since Starlette 0.20.
- Use class-based `HTTPEndpoint` for structured method dispatch (get/post/put/delete).
- Use `BackgroundTasks` for fire-and-forget I/O after returning a response.
- Use `StreamingResponse` with `text/event-stream` for SSE — set `X-Accel-Buffering: no` for nginx.
- Middleware ordering matters: add security/CORS middleware before logging middleware.
- All endpoints and middleware must be `async` — Starlette is async-native.
