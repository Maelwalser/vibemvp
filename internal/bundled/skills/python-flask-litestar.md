# Python + Flask / Litestar Skill Guide

## Project Layout

```
service-name/
├── pyproject.toml
├── wsgi.py              # Flask entry: app = create_app()
├── app/
│   ├── __init__.py      # create_app() factory
│   ├── blueprints/      # Flask Blueprints / Litestar Controllers
│   ├── models.py        # SQLAlchemy / dataclass models
│   ├── schemas.py       # Marshmallow / dataclass DTOs
│   ├── services/
│   └── config.py
└── tests/
```

## Dependencies

```toml
[project]
dependencies = [
    # Flask stack
    "flask>=3.0.0",
    "flask-sqlalchemy>=3.1.0",
    "flask-migrate>=4.0.0",
    # OR Litestar stack
    "litestar>=2.8.0",
    "uvicorn[standard]>=0.29.0",
    "advanced-alchemy>=0.9.0",
]
```

---

## Flask: App Factory Pattern

```python
# app/__init__.py
from flask import Flask
from flask_sqlalchemy import SQLAlchemy
from .config import Config

db = SQLAlchemy()

def create_app(config: type = Config) -> Flask:
    app = Flask(__name__)
    app.config.from_object(config)

    db.init_app(app)

    from .blueprints.users import users_bp
    from .blueprints.items import items_bp
    app.register_blueprint(users_bp, url_prefix="/users")
    app.register_blueprint(items_bp, url_prefix="/items")

    @app.get("/health")
    def health():
        return {"status": "ok"}

    return app
```

## Flask: Blueprint

```python
# app/blueprints/users.py
from flask import Blueprint, request, jsonify, abort
from app import db
from app.models import User

users_bp = Blueprint("users", __name__)

@users_bp.post("/")
def create_user():
    data = request.get_json(force=True, silent=True)
    if not data:
        abort(400, description="Invalid JSON body")
    user = User(name=data["name"], email=data["email"])
    db.session.add(user)
    db.session.commit()
    return jsonify(user.to_dict()), 201

@users_bp.get("/<int:user_id>")
def get_user(user_id: int):
    user = db.session.get(User, user_id)
    if user is None:
        abort(404, description="User not found")
    return jsonify(user.to_dict())
```

## Flask: Error Handlers

```python
# app/__init__.py (inside create_app)
from flask import jsonify

@app.errorhandler(400)
def bad_request(e):
    return jsonify({"error": str(e.description)}), 400

@app.errorhandler(404)
def not_found(e):
    return jsonify({"error": str(e.description)}), 404

@app.errorhandler(500)
def internal_error(e):
    app.logger.exception("Unhandled error")
    return jsonify({"error": "internal server error"}), 500
```

---

## Litestar: Controller + DTO

```python
# app/controllers/users.py
from dataclasses import dataclass
from litestar import Controller, get, post
from litestar.di import Provide
from litestar.dto import DataclassDTO
from app.services.user_service import UserService

@dataclass
class UserCreate:
    name: str
    email: str

@dataclass
class UserResponse:
    id: int
    name: str
    email: str

UserCreateDTO = DataclassDTO[UserCreate]
UserResponseDTO = DataclassDTO[UserResponse]

class UserController(Controller):
    path = "/users"
    dependencies = {"svc": Provide(lambda: UserService())}

    @post("/", dto=UserCreateDTO, return_dto=UserResponseDTO)
    async def create_user(self, data: UserCreate, svc: UserService) -> UserResponse:
        return await svc.create(data)

    @get("/{user_id:int}", return_dto=UserResponseDTO)
    async def get_user(self, user_id: int, svc: UserService) -> UserResponse:
        return await svc.get(user_id)
```

## Litestar: App Setup

```python
# app/__init__.py
from litestar import Litestar
from litestar.openapi import OpenAPIConfig
from app.controllers.users import UserController

app = Litestar(
    route_handlers=[UserController],
    openapi_config=OpenAPIConfig(title="My Service", version="1.0.0"),
)
```

## Error Handling

**Flask:**
- Use `abort(status_code, description="...")` for HTTP errors.
- Register `@app.errorhandler(code)` handlers to return consistent JSON.
- Never let exceptions propagate unhandled to the client.

**Litestar:**
- Raise `litestar.exceptions.HTTPException(status_code=..., detail="...")`.
- Use `exception_handlers` dict on `Litestar(...)` for global handlers.

## Key Rules

- Flask: Always use the app factory `create_app()` — never a module-level `app = Flask(...)` in production code.
- Flask: Organize routes with `Blueprint` — never add routes directly to the app object outside the factory.
- Flask: Use `request.get_json(silent=True)` and validate before use; never trust raw input.
- Litestar: Use `DataclassDTO` or `MsgspecDTO` for request/response typing — never return raw dicts from handlers.
- Litestar: Use `Provide()` in `dependencies` for DI — do not instantiate services inside handlers.
- Both: Read config exclusively from environment variables or a config object — no hardcoded values.
