# Rust + Rocket & Warp Skill Guide

## Project Layout

```
service-name/
├── Cargo.toml
├── Rocket.toml          # Rocket: environment config
├── src/
│   ├── main.rs          # Entry point
│   ├── routes/          # Route handlers
│   ├── guards/          # Rocket: FromRequest guards
│   ├── models/          # Request/response types
│   ├── services/        # Business logic
│   └── filters/         # Warp: filter definitions
└── tests/
```

## Cargo.toml

```toml
[dependencies]
# Rocket
rocket = { version = "0.5", features = ["json"] }

# Warp (alternative)
warp = "0.3"
tokio = { version = "1", features = ["full"] }

# Shared
serde = { version = "1", features = ["derive"] }
serde_json = "1"
thiserror = "1"
```

---

## Rocket

### Server Setup

```rust
// src/main.rs (Rocket)
#[macro_use] extern crate rocket;

use rocket::{Build, Rocket};

#[launch]
fn rocket() -> Rocket<Build> {
    rocket::build()
        .attach(DbFairing)          // fairing: lifecycle hooks
        .manage(AppConfig::from_env())  // managed state
        .mount("/api", routes![
            routes::users::list,
            routes::users::get_by_id,
            routes::users::create,
            routes::users::delete,
        ])
        .mount("/", routes![routes::health::health])
}
```

### Route Macros

```rust
// src/routes/users.rs
use rocket::{serde::json::Json, State, http::Status};

#[get("/users")]
pub async fn list(
    db: &State<DbPool>,
) -> Result<Json<Vec<UserResponse>>, AppError> {
    let users = UserService::list(db.inner()).await?;
    Ok(Json(users))
}

#[get("/users/<id>")]
pub async fn get_by_id(
    id: i64,
    db: &State<DbPool>,
) -> Result<Json<UserResponse>, AppError> {
    let user = UserService::find_by_id(db.inner(), id)
        .await?
        .ok_or(AppError::NotFound)?;
    Ok(Json(user))
}

#[post("/users", data = "<request>")]
pub async fn create(
    request: Json<CreateUserRequest>,
    db: &State<DbPool>,
) -> Result<(Status, Json<UserResponse>), AppError> {
    let user = UserService::create(db.inner(), request.into_inner()).await?;
    Ok((Status::Created, Json(user)))
}

#[delete("/users/<id>")]
pub async fn delete(
    id: i64,
    db: &State<DbPool>,
) -> Result<Status, AppError> {
    UserService::delete(db.inner(), id).await?;
    Ok(Status::NoContent)
}
```

### FromRequest Guards

```rust
// src/guards/auth.rs
use rocket::{request::{FromRequest, Outcome, Request}, http::Status};

pub struct AuthenticatedUser {
    pub user_id: i64,
}

#[rocket::async_trait]
impl<'r> FromRequest<'r> for AuthenticatedUser {
    type Error = String;

    async fn from_request(request: &'r Request<'_>) -> Outcome<Self, Self::Error> {
        let token = request.headers().get_one("Authorization")
            .and_then(|v| v.strip_prefix("Bearer "));

        match token {
            Some(t) => match validate_jwt(t) {
                Ok(user_id) => Outcome::Success(AuthenticatedUser { user_id }),
                Err(e) => Outcome::Error((Status::Unauthorized, e.to_string())),
            },
            None => Outcome::Error((Status::Unauthorized, "Missing token".to_string())),
        }
    }
}

// Usage in route — guard is injected as a parameter:
#[get("/me")]
pub async fn get_me(user: AuthenticatedUser, db: &State<DbPool>) -> Result<Json<UserResponse>, AppError> {
    let me = UserService::find_by_id(db.inner(), user.user_id).await?
        .ok_or(AppError::NotFound)?;
    Ok(Json(me))
}
```

### Responder Trait

```rust
use rocket::{http::ContentType, response::{self, Responder, Response}, Request};
use std::io::Cursor;

pub struct HtmlPage(pub String);

impl<'r> Responder<'r, 'static> for HtmlPage {
    fn respond_to(self, _: &'r Request<'_>) -> response::Result<'static> {
        Response::build()
            .header(ContentType::HTML)
            .sized_body(self.0.len(), Cursor::new(self.0))
            .ok()
    }
}
```

### Fairing Hooks

```rust
use rocket::{fairing::{Fairing, Info, Kind}, Rocket, Build};

pub struct DbFairing;

#[rocket::async_trait]
impl Fairing for DbFairing {
    fn info(&self) -> Info {
        Info { name: "Database Pool", kind: Kind::Ignite }
    }

    async fn on_ignite(&self, rocket: Rocket<Build>) -> rocket::fairing::Result {
        let database_url = std::env::var("DATABASE_URL").expect("DATABASE_URL required");
        match DbPool::connect(&database_url).await {
            Ok(pool) => Ok(rocket.manage(pool)),
            Err(e) => {
                eprintln!("Failed to connect to database: {e}");
                Err(rocket)
            }
        }
    }
}
```

### Rocket.toml

```toml
[default]
address = "0.0.0.0"
port = 8080
log_level = "normal"

[release]
log_level = "critical"
secret_key = "${ROCKET_SECRET_KEY}"
```

---

## Warp

### Server Setup

```rust
// src/main.rs (Warp)
use warp::Filter;

#[tokio::main]
async fn main() {
    let db = DbPool::connect(&std::env::var("DATABASE_URL").unwrap()).await.unwrap();
    let db = std::sync::Arc::new(db);

    let routes = filters::users(db.clone())
        .or(filters::health())
        .with(warp::log("service"));

    let port: u16 = std::env::var("PORT")
        .unwrap_or_else(|_| "8080".to_string())
        .parse()
        .unwrap();

    warp::serve(routes).run(([0, 0, 0, 0], port)).await;
}
```

### Filter Composition

```rust
// src/filters/users.rs
use warp::{Filter, Rejection, Reply};
use std::sync::Arc;

pub fn users(db: Arc<DbPool>) -> impl Filter<Extract = impl Reply, Error = Rejection> + Clone {
    list_users(db.clone())
        .or(get_user(db.clone()))
        .or(create_user(db.clone()))
        .or(delete_user(db.clone()))
}

// Shared state filter — injects db into handlers
fn with_db(db: Arc<DbPool>) -> impl Filter<Extract = (Arc<DbPool>,), Error = std::convert::Infallible> + Clone {
    warp::any().map(move || db.clone())
}

fn list_users(db: Arc<DbPool>) -> impl Filter<Extract = impl Reply, Error = Rejection> + Clone {
    warp::path("users")
        .and(warp::get())
        .and(warp::query::<ListQuery>())
        .and(with_db(db))
        .and_then(handlers::list_users)
}

fn get_user(db: Arc<DbPool>) -> impl Filter<Extract = impl Reply, Error = Rejection> + Clone {
    warp::path!("users" / i64)
        .and(warp::get())
        .and(with_db(db))
        .and_then(handlers::get_user)
}

fn create_user(db: Arc<DbPool>) -> impl Filter<Extract = impl Reply, Error = Rejection> + Clone {
    warp::path("users")
        .and(warp::post())
        .and(warp::body::json())
        .and(with_db(db))
        .and_then(handlers::create_user)
}

fn delete_user(db: Arc<DbPool>) -> impl Filter<Extract = impl Reply, Error = Rejection> + Clone {
    warp::path!("users" / i64)
        .and(warp::delete())
        .and(with_db(db))
        .and_then(handlers::delete_user)
}
```

### Warp Handlers

```rust
// src/handlers/users.rs
use warp::{reject, reply, Rejection, Reply};
use std::sync::Arc;

pub async fn list_users(
    query: ListQuery,
    db: Arc<DbPool>,
) -> Result<impl Reply, Rejection> {
    let users = UserService::list(&db, query.page.unwrap_or(0), query.limit.unwrap_or(20))
        .await
        .map_err(|e| reject::custom(AppError::from(e)))?;
    Ok(reply::json(&users))
}

pub async fn get_user(id: i64, db: Arc<DbPool>) -> Result<impl Reply, Rejection> {
    let user = UserService::find_by_id(&db, id)
        .await
        .map_err(|e| reject::custom(AppError::from(e)))?
        .ok_or_else(|| reject::custom(AppError::NotFound))?;
    Ok(reply::json(&user))
}

pub async fn create_user(
    body: CreateUserRequest,
    db: Arc<DbPool>,
) -> Result<impl Reply, Rejection> {
    let user = UserService::create(&db, body)
        .await
        .map_err(|e| reject::custom(AppError::from(e)))?;
    Ok(reply::with_status(reply::json(&user), warp::http::StatusCode::CREATED))
}
```

### Warp Rejection Handling

```rust
use warp::{Rejection, Reply, http::StatusCode};
use serde_json::json;

// Custom rejection type
#[derive(Debug)]
pub enum AppError {
    NotFound,
    Unauthorized,
    Internal(String),
}

impl warp::reject::Reject for AppError {}

// Global rejection handler
pub async fn handle_rejection(err: Rejection) -> Result<impl Reply, std::convert::Infallible> {
    let (status, message) = if err.is_not_found() {
        (StatusCode::NOT_FOUND, "Not found")
    } else if let Some(e) = err.find::<AppError>() {
        match e {
            AppError::NotFound => (StatusCode::NOT_FOUND, "Resource not found"),
            AppError::Unauthorized => (StatusCode::UNAUTHORIZED, "Unauthorized"),
            AppError::Internal(_) => (StatusCode::INTERNAL_SERVER_ERROR, "Internal server error"),
        }
    } else {
        (StatusCode::INTERNAL_SERVER_ERROR, "Internal server error")
    };

    Ok(reply::with_status(
        reply::json(&json!({"error": message})),
        status,
    ))
}

// Wire in main:
// let routes = api_routes().recover(handle_rejection);
```

## Rules

**Rocket:**
- Use `#[get("/<param>")]` macros for routes — Rocket validates path parameter types at compile time.
- Use `FromRequest` guards for auth, rate limiting, and request validation — composable and reusable.
- Use `State<T>` to inject managed state; call `db.inner()` to get the reference inside the guard.
- Wire lifecycle logic in `Fairing` implementations — do not run setup code in `rocket()`.
- Use `Rocket.toml` for per-environment config; override with env vars prefixed `ROCKET_`.

**Warp:**
- Warp filters compose with `.and()` — each `.and()` adds one more extracted value as a handler argument.
- Use `warp::path!("users" / i64)` for typed path segments — they must match handler argument order.
- Implement `warp::reject::Reject` on custom error types; use `.recover(handle_rejection)` globally.
- State injection uses a shared filter `fn with_db(...) -> impl Filter<Extract = (Arc<DbPool>,), ...>`.
- Use `reply::with_status(reply::json(&body), StatusCode::CREATED)` for non-200 responses.
