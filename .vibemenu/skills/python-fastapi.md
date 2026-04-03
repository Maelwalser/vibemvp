# Python + FastAPI Skill Guide

## Project Layout

```
service-name/
├── pyproject.toml
├── main.py
├── app/
│   ├── __init__.py
│   ├── routers/         # APIRouter modules
│   ├── models/          # Pydantic schemas
│   ├── services/        # Business logic
│   ├── repositories/    # Data access
│   ├── dependencies.py  # Shared Depends() providers
│   └── config.py        # Settings via pydantic-settings
└── tests/
```

## Dependencies

```toml
[project]
dependencies = [
    "fastapi>=0.111.0",
    "uvicorn[standard]>=0.29.0",
    "pydantic>=2.7.0",
    "pydantic-settings>=2.2.0",
    "sqlalchemy>=2.0.0",
    "asyncpg>=0.29.0",
]
```

## Server Setup with Lifespan

```python
# main.py
from contextlib import asynccontextmanager
from fastapi import FastAPI
from app.routers import users, items
from app.database import engine

@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    await engine.connect()
    yield
    # Shutdown
    await engine.dispose()

app = FastAPI(title="My Service", lifespan=lifespan)
app.include_router(users.router, prefix="/users", tags=["users"])
app.include_router(items.router, prefix="/items", tags=["items"])

@app.get("/health")
async def health() -> dict:
    return {"status": "ok"}
```

## Pydantic Schemas

```python
# app/models/user.py
from pydantic import BaseModel, EmailStr, field_validator
from datetime import datetime

class UserCreate(BaseModel):
    name: str
    email: EmailStr
    age: int

    @field_validator("age")
    @classmethod
    def age_must_be_positive(cls, v: int) -> int:
        if v < 0:
            raise ValueError("age must be positive")
        return v

class UserResponse(BaseModel):
    id: int
    name: str
    email: str
    created_at: datetime

    model_config = {"from_attributes": True}
```

## Modular Routing with APIRouter

```python
# app/routers/users.py
from fastapi import APIRouter, Depends, HTTPException, status
from app.models.user import UserCreate, UserResponse
from app.dependencies import get_user_service
from app.services.user_service import UserService

router = APIRouter()

@router.post("/", response_model=UserResponse, status_code=status.HTTP_201_CREATED)
async def create_user(
    body: UserCreate,
    svc: UserService = Depends(get_user_service),
) -> UserResponse:
    return await svc.create(body)

@router.get("/{user_id}", response_model=UserResponse)
async def get_user(
    user_id: int,
    svc: UserService = Depends(get_user_service),
) -> UserResponse:
    user = await svc.get(user_id)
    if user is None:
        raise HTTPException(status_code=404, detail="User not found")
    return user
```

## Dependency Injection with Depends()

```python
# app/dependencies.py
from fastapi import Depends
from sqlalchemy.ext.asyncio import AsyncSession
from app.database import get_session
from app.repositories.user_repo import UserRepository
from app.services.user_service import UserService

async def get_user_repo(
    session: AsyncSession = Depends(get_session),
) -> UserRepository:
    return UserRepository(session)

async def get_user_service(
    repo: UserRepository = Depends(get_user_repo),
) -> UserService:
    return UserService(repo)
```

## Settings

```python
# app/config.py
from pydantic_settings import BaseSettings, SettingsConfigDict

class Settings(BaseSettings):
    database_url: str
    secret_key: str
    debug: bool = False

    model_config = SettingsConfigDict(env_file=".env", env_file_encoding="utf-8")

settings = Settings()
```

## Error Handling

- Raise `HTTPException(status_code=..., detail="...")` for HTTP errors.
- Add custom exception handlers with `@app.exception_handler(MyError)`.
- Never swallow exceptions silently — log and re-raise or convert to HTTPException.

## Key Rules

- All route functions must be `async def` for non-blocking I/O.
- Use `response_model=` on every route to enforce output schema and strip extra fields.
- Put shared logic in `Depends()` providers, not inside route functions.
- Use `pydantic-settings` for config — never hardcode secrets.
- OpenAPI docs are auto-generated at `/docs` (Swagger) and `/redoc`.
- Use `status_code=` explicitly on POST (201) and DELETE (204) routes.
