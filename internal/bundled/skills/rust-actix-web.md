# Rust + Actix-web Skill Guide

## Project Layout

```
service-name/
├── Cargo.toml
├── src/
│   ├── main.rs          # Entry point, HttpServer setup, app factory
│   ├── config.rs        # Config from environment
│   ├── routes/
│   │   ├── mod.rs       # Route registration helper
│   │   ├── users.rs     # User handlers
│   │   └── health.rs    # Health check
│   ├── models/          # Serde request/response types
│   ├── services/        # Business logic
│   └── errors.rs        # ResponseError implementations
└── tests/               # Integration tests
```

## Cargo.toml

```toml
[package]
name = "service-name"
version = "0.1.0"
edition = "2021"

[dependencies]
actix-web = "4"
actix-rt = "2"
serde = { version = "1", features = ["derive"] }
serde_json = "1"
tokio = { version = "1", features = ["full"] }
sqlx = { version = "0.7", features = ["postgres", "runtime-tokio-rustls", "uuid"] }
uuid = { version = "1", features = ["serde", "v4"] }
thiserror = "1"
tracing = "0.1"
tracing-actix-web = "0.7"
tracing-subscriber = { version = "0.3", features = ["env-filter"] }

[dev-dependencies]
actix-web = { version = "4", features = ["macros"] }
```

## Server Setup

```rust
// src/main.rs
use actix_web::{middleware, web, App, HttpServer};
use sqlx::PgPool;
use std::sync::Arc;

#[derive(Clone)]
pub struct AppState {
    pub db: PgPool,
    pub config: Arc<AppConfig>,
}

pub struct AppConfig {
    pub jwt_secret: String,
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(tracing_subscriber::EnvFilter::from_default_env())
        .init();

    let database_url = std::env::var("DATABASE_URL").expect("DATABASE_URL must be set");
    let db = PgPool::connect(&database_url).await.expect("Failed to connect to DB");

    let state = AppState {
        db,
        config: Arc::new(AppConfig {
            jwt_secret: std::env::var("JWT_SECRET").expect("JWT_SECRET must be set"),
        }),
    };

    let port = std::env::var("PORT").unwrap_or_else(|_| "8080".to_string());
    tracing::info!("Starting server on port {port}");

    HttpServer::new(move || {
        App::new()
            .app_data(web::Data::new(state.clone()))
            .wrap(tracing_actix_web::TracingLogger::default())
            .wrap(middleware::Compress::default())
            .configure(routes::users::configure)
            .configure(routes::health::configure)
    })
    .bind(format!("0.0.0.0:{port}"))?
    .run()
    .await
}
```

## Handler Pattern

```rust
// src/routes/users.rs
use actix_web::{web, HttpResponse, Responder};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

pub fn configure(cfg: &mut web::ServiceConfig) {
    cfg.service(
        web::scope("/users")
            .route("", web::get().to(list_users))
            .route("", web::post().to(create_user))
            .route("/{id}", web::get().to(get_user))
            .route("/{id}", web::delete().to(delete_user)),
    );
}

// Macro-style route alternative:
#[actix_web::get("/users")]
async fn list_users_macro(
    data: web::Data<AppState>,
    query: web::Query<ListQuery>,
) -> Result<impl Responder, AppError> {
    let users = UserService::list(&data.db, query.page, query.limit).await?;
    Ok(HttpResponse::Ok().json(users))
}

#[derive(Deserialize)]
pub struct ListQuery {
    page: Option<u32>,
    limit: Option<u32>,
}

#[derive(Deserialize)]
pub struct CreateUserRequest {
    pub name: String,
    pub email: String,
}

#[derive(Serialize)]
pub struct UserResponse {
    pub id: Uuid,
    pub name: String,
    pub email: String,
}

async fn list_users(
    data: web::Data<AppState>,
    query: web::Query<ListQuery>,
) -> Result<HttpResponse, AppError> {
    let page = query.page.unwrap_or(0);
    let limit = query.limit.unwrap_or(20).min(100);
    let users = UserService::list(&data.db, page, limit).await?;
    Ok(HttpResponse::Ok().json(users))
}

async fn get_user(
    data: web::Data<AppState>,
    path: web::Path<Uuid>,
) -> Result<HttpResponse, AppError> {
    let id = path.into_inner();
    let user = UserService::find_by_id(&data.db, id)
        .await?
        .ok_or(AppError::NotFound)?;
    Ok(HttpResponse::Ok().json(user))
}

async fn create_user(
    data: web::Data<AppState>,
    body: web::Json<CreateUserRequest>,
) -> Result<HttpResponse, AppError> {
    let user = UserService::create(&data.db, body.into_inner()).await?;
    Ok(HttpResponse::Created().json(user))
}

async fn delete_user(
    data: web::Data<AppState>,
    path: web::Path<Uuid>,
) -> Result<HttpResponse, AppError> {
    let id = path.into_inner();
    UserService::delete(&data.db, id).await?;
    Ok(HttpResponse::NoContent().finish())
}
```

## Data<T> for Shared State

```rust
// web::Data<T> is Arc<T> under the hood — Clone is cheap
// State must implement Send + Sync for multi-threaded workers

// Inject in handler:
async fn my_handler(
    state: web::Data<AppState>,
) -> impl Responder {
    // state.db is accessible here
    HttpResponse::Ok().finish()
}

// Multiple state types can coexist:
App::new()
    .app_data(web::Data::new(db_pool))
    .app_data(web::Data::new(redis_client))
    .app_data(web::Data::new(config))
```

## Custom Error Type with ResponseError

```rust
// src/errors.rs
use actix_web::{HttpResponse, ResponseError};
use serde_json::json;
use thiserror::Error;

#[derive(Debug, Error)]
pub enum AppError {
    #[error("Resource not found")]
    NotFound,
    #[error("Validation error: {0}")]
    Validation(String),
    #[error("Unauthorized")]
    Unauthorized,
    #[error("Database error")]
    Database(#[from] sqlx::Error),
}

impl ResponseError for AppError {
    fn status_code(&self) -> actix_web::http::StatusCode {
        match self {
            AppError::NotFound => actix_web::http::StatusCode::NOT_FOUND,
            AppError::Validation(_) => actix_web::http::StatusCode::BAD_REQUEST,
            AppError::Unauthorized => actix_web::http::StatusCode::UNAUTHORIZED,
            AppError::Database(e) => {
                tracing::error!("Database error: {:?}", e);
                actix_web::http::StatusCode::INTERNAL_SERVER_ERROR
            }
        }
    }

    fn error_response(&self) -> HttpResponse {
        let body = match self {
            AppError::Database(_) => json!({"error": "Internal server error"}),
            other => json!({"error": other.to_string()}),
        };
        HttpResponse::build(self.status_code()).json(body)
    }
}
```

## Actix-web vs Raw Errors

```rust
// Use actix_web::Error only for framework-level errors
// Use your own AppError (implementing ResponseError) for domain errors

// Wrapping actix errors:
async fn handler() -> Result<HttpResponse, actix_web::Error> {
    let body = web::Json::<MyStruct>::from_request(...)
        .await
        .map_err(|e| actix_web::error::ErrorBadRequest(e))?;
    Ok(HttpResponse::Ok().json(body))
}
```

## Middleware and App Configuration

```rust
use actix_web::middleware::{Compress, DefaultHeaders, Logger};

App::new()
    .app_data(web::Data::new(state.clone()))
    .wrap(tracing_actix_web::TracingLogger::default())
    .wrap(Compress::default())
    .wrap(
        DefaultHeaders::new()
            .add(("X-Content-Type-Options", "nosniff"))
            .add(("X-Frame-Options", "DENY")),
    )
    .app_data(
        web::JsonConfig::default()
            .limit(1_048_576)  // 1 MB body limit
            .error_handler(|err, _| {
                actix_web::error::InternalError::from_response(
                    err,
                    HttpResponse::BadRequest().json(json!({"error": "Invalid JSON"})),
                )
                .into()
            }),
    )
```

## Multithreading Constraints

Actix-web runs multiple worker threads. All state in `web::Data<T>` must be `Send + Sync`:
- Wrap non-Send types in `Arc<Mutex<T>>` or `Arc<RwLock<T>>`
- `PgPool` from sqlx is already `Send + Sync + Clone`
- Avoid `Rc<T>` and `Cell<T>` in handler state

```rust
use std::sync::{Arc, RwLock};

pub struct AppState {
    pub db: PgPool,
    pub cache: Arc<RwLock<HashMap<String, String>>>,  // thread-safe mutable state
}
```

## Health Check

```rust
pub fn configure(cfg: &mut web::ServiceConfig) {
    cfg.service(
        web::scope("")
            .route("/health", web::get().to(health))
            .route("/ready", web::get().to(ready)),
    );
}

async fn health() -> impl Responder {
    HttpResponse::Ok().json(json!({"status": "ok"}))
}

async fn ready(data: web::Data<AppState>) -> impl Responder {
    match sqlx::query("SELECT 1").fetch_one(&data.db).await {
        Ok(_) => HttpResponse::Ok().json(json!({"status": "ready"})),
        Err(_) => HttpResponse::ServiceUnavailable().json(json!({"status": "unavailable"})),
    }
}
```

## Rules

- Use `web::Data<AppState>` for shared state — it is `Arc<AppState>` internally, so `Clone` is cheap.
- All state in `web::Data<T>` must implement `Send + Sync` — wrap non-thread-safe types in `Arc<Mutex<T>>`.
- Implement `ResponseError` on custom error types — return `Result<HttpResponse, AppError>` from handlers.
- Never leak internal error details (DB errors, stack traces) in HTTP responses — log them server-side.
- Use `web::ServiceConfig` for route organization — group related routes in a `configure` function per module.
- Use `tracing-actix-web` for structured request logging instead of the built-in `Logger` middleware.
- Configure `web::JsonConfig` with a reasonable body size limit and custom error handler.
- Use `#[actix_web::main]` on `main` for the Actix runtime; use `#[actix_web::test]` for integration tests.
