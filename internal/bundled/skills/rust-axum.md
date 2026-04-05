# Rust + Axum Skill Guide

## Project Layout

```
service-name/
├── Cargo.toml
├── src/
│   ├── main.rs          # Entry point, router setup, graceful shutdown
│   ├── state.rs         # AppState definition
│   ├── routes/
│   │   ├── mod.rs       # Route registration
│   │   ├── users.rs     # User handlers
│   │   └── health.rs    # Health check
│   ├── models/          # Request/response types
│   ├── services/        # Business logic
│   └── errors.rs        # Error types implementing IntoResponse
└── tests/               # Integration tests
```

## Cargo.toml

```toml
[package]
name = "service-name"
version = "0.1.0"
edition = "2021"

[dependencies]
axum = { version = "0.7", features = ["macros"] }
tokio = { version = "1", features = ["full"] }
tower = { version = "0.4", features = ["full"] }
tower-http = { version = "0.5", features = ["cors", "trace", "compression-gzip"] }
serde = { version = "1", features = ["derive"] }
serde_json = "1"
sqlx = { version = "0.7", features = ["postgres", "runtime-tokio-rustls", "uuid", "chrono"] }
uuid = { version = "1", features = ["serde", "v4"] }
tracing = "0.1"
tracing-subscriber = { version = "0.3", features = ["env-filter"] }
thiserror = "1"
anyhow = "1"

[dev-dependencies]
axum-test = "14"
```

## App State

```rust
// src/state.rs
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
```

## Server Setup & Graceful Shutdown

```rust
// src/main.rs
use axum::{Router, serve};
use tokio::net::TcpListener;
use tokio::signal;
use tower_http::trace::TraceLayer;
use tower_http::cors::CorsLayer;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(tracing_subscriber::EnvFilter::from_default_env())
        .init();

    let database_url = std::env::var("DATABASE_URL").expect("DATABASE_URL must be set");
    let db = sqlx::PgPool::connect(&database_url).await?;

    let state = AppState {
        db,
        config: Arc::new(AppConfig {
            jwt_secret: std::env::var("JWT_SECRET").expect("JWT_SECRET must be set"),
        }),
    };

    let app = Router::new()
        .merge(routes::users::router())
        .merge(routes::health::router())
        .layer(TraceLayer::new_for_http())
        .layer(CorsLayer::permissive())  // restrict in production
        .with_state(state);

    let port = std::env::var("PORT").unwrap_or_else(|_| "8080".to_string());
    let listener = TcpListener::bind(format!("0.0.0.0:{port}")).await?;
    tracing::info!("Listening on {}", listener.local_addr()?);

    serve(listener, app)
        .with_graceful_shutdown(shutdown_signal())
        .await?;

    Ok(())
}

async fn shutdown_signal() {
    let ctrl_c = async {
        signal::ctrl_c().await.expect("failed to install Ctrl+C handler");
    };

    #[cfg(unix)]
    let terminate = async {
        signal::unix::signal(signal::unix::SignalKind::terminate())
            .expect("failed to install signal handler")
            .recv()
            .await;
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {},
        _ = terminate => {},
    }
    tracing::info!("Shutdown signal received");
}
```

## Handler Pattern

```rust
// src/routes/users.rs
use axum::{
    extract::{Path, Query, State},
    http::StatusCode,
    response::IntoResponse,
    routing::{delete, get, post},
    Json, Router,
};
use serde::{Deserialize, Serialize};

pub fn router() -> Router<AppState> {
    Router::new()
        .route("/users", get(list_users).post(create_user))
        .route("/users/:id", get(get_user).delete(delete_user))
}

#[derive(Debug, Deserialize)]
pub struct ListQuery {
    page: Option<u32>,
    limit: Option<u32>,
}

#[derive(Debug, Deserialize)]
pub struct CreateUserRequest {
    pub name: String,
    pub email: String,
}

#[derive(Debug, Serialize)]
pub struct UserResponse {
    pub id: uuid::Uuid,
    pub name: String,
    pub email: String,
}

async fn list_users(
    State(state): State<AppState>,
    Query(params): Query<ListQuery>,
) -> Result<Json<Vec<UserResponse>>, AppError> {
    let page = params.page.unwrap_or(0);
    let limit = params.limit.unwrap_or(20).min(100);
    let users = UserService::list(&state.db, page, limit).await?;
    Ok(Json(users))
}

async fn get_user(
    State(state): State<AppState>,
    Path(id): Path<uuid::Uuid>,
) -> Result<Json<UserResponse>, AppError> {
    let user = UserService::find_by_id(&state.db, id)
        .await?
        .ok_or(AppError::NotFound)?;
    Ok(Json(user))
}

async fn create_user(
    State(state): State<AppState>,
    Json(payload): Json<CreateUserRequest>,
) -> Result<(StatusCode, Json<UserResponse>), AppError> {
    let user = UserService::create(&state.db, payload).await?;
    Ok((StatusCode::CREATED, Json(user)))
}

async fn delete_user(
    State(state): State<AppState>,
    Path(id): Path<uuid::Uuid>,
) -> Result<StatusCode, AppError> {
    UserService::delete(&state.db, id).await?;
    Ok(StatusCode::NO_CONTENT)
}
```

## Error Type Implementing IntoResponse

```rust
// src/errors.rs
use axum::{http::StatusCode, response::{IntoResponse, Response}, Json};
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
    #[error("Database error: {0}")]
    Database(#[from] sqlx::Error),
    #[error("Internal error: {0}")]
    Internal(#[from] anyhow::Error),
}

impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let (status, message) = match &self {
            AppError::NotFound => (StatusCode::NOT_FOUND, self.to_string()),
            AppError::Validation(msg) => (StatusCode::BAD_REQUEST, msg.clone()),
            AppError::Unauthorized => (StatusCode::UNAUTHORIZED, self.to_string()),
            AppError::Database(e) => {
                tracing::error!("Database error: {:?}", e);
                (StatusCode::INTERNAL_SERVER_ERROR, "Database error".to_string())
            }
            AppError::Internal(e) => {
                tracing::error!("Internal error: {:?}", e);
                (StatusCode::INTERNAL_SERVER_ERROR, "Internal server error".to_string())
            }
        };
        (status, Json(json!({"error": message}))).into_response()
    }
}
```

## Tower Middleware Layers

```rust
use tower::ServiceBuilder;
use tower_http::{
    compression::CompressionLayer,
    cors::CorsLayer,
    trace::TraceLayer,
    timeout::TimeoutLayer,
};
use std::time::Duration;

let app = Router::new()
    .merge(api_routes())
    .layer(
        ServiceBuilder::new()
            .layer(TraceLayer::new_for_http())
            .layer(CompressionLayer::new())
            .layer(TimeoutLayer::new(Duration::from_secs(30)))
            .layer(
                CorsLayer::new()
                    .allow_origin("https://example.com".parse::<HeaderValue>().unwrap())
                    .allow_methods([Method::GET, Method::POST, Method::PUT, Method::DELETE])
                    .allow_headers([CONTENT_TYPE, AUTHORIZATION]),
            ),
    )
    .with_state(state);
```

## Health Check

```rust
// src/routes/health.rs
pub fn router() -> Router<AppState> {
    Router::new()
        .route("/health", get(health))
        .route("/ready", get(ready))
}

async fn health() -> impl IntoResponse {
    Json(serde_json::json!({"status": "ok"}))
}

async fn ready(State(state): State<AppState>) -> impl IntoResponse {
    // Check DB connectivity
    match sqlx::query("SELECT 1").fetch_one(&state.db).await {
        Ok(_) => Json(serde_json::json!({"status": "ready"})),
        Err(_) => (StatusCode::SERVICE_UNAVAILABLE, Json(serde_json::json!({"status": "unavailable"}))).into_response(),
    }
}
```

## Rules

- `State<AppState>` extractor requires `AppState: Clone` — wrap non-Clone fields in `Arc<T>`.
- Extractors are consumed in order: `State` before `Path`/`Query` before `Json` (body last).
- Implement `IntoResponse` on custom error types — never return raw strings from handlers.
- Use `thiserror` for error type definitions; use `anyhow` for error propagation in application code.
- Use `tracing` (not `println!`) for all logging; configure with `RUST_LOG` env var.
- Graceful shutdown prevents in-flight requests from being dropped — always implement it in production.
- Keep handlers thin: extract logic to service functions that take `&PgPool` and return `Result<T, AppError>`.
- Use `tower-http` layers for CORS, compression, tracing — do not implement these manually.
