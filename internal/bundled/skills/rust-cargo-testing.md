# Rust + Cargo Testing Skill Guide

## Test Organization

```
service-name/
├── src/
│   ├── lib.rs           # Library crate root
│   ├── models/
│   │   └── user.rs      # Unit tests in #[cfg(test)] modules
│   └── services/
│       └── user_service.rs
└── tests/               # Integration tests — separate crate, no access to private items
    ├── users_api.rs
    └── common/
        └── mod.rs       # Shared test helpers
```

## Unit Tests with #[cfg(test)]

```rust
// src/services/user_service.rs
pub struct UserService;

impl UserService {
    pub fn validate_email(email: &str) -> bool {
        email.contains('@') && email.contains('.')
    }

    pub fn format_name(first: &str, last: &str) -> String {
        format!("{} {}", first.trim(), last.trim())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn validate_email_accepts_valid_address() {
        assert!(UserService::validate_email("user@example.com"));
    }

    #[test]
    fn validate_email_rejects_missing_at() {
        assert!(!UserService::validate_email("userexample.com"));
    }

    #[test]
    fn format_name_trims_whitespace() {
        let result = UserService::format_name("  Alice ", " Smith  ");
        assert_eq!(result, "Alice Smith");
    }

    #[test]
    #[should_panic(expected = "index out of bounds")]
    fn panics_on_invalid_access() {
        let v: Vec<i32> = vec![];
        let _ = v[0]; // intentional panic for demo
    }
}
```

## assert! Macros

```rust
#[test]
fn assertion_examples() {
    // Basic assertion
    assert!(1 + 1 == 2);
    assert!(value.is_some(), "Expected Some but got None");

    // Equality with diff output on failure
    assert_eq!(result, expected);
    assert_eq!(result, expected, "Custom failure message: got {result}");

    // Inequality
    assert_ne!(result, wrong_value);

    // Approximate float equality (no built-in — use a delta):
    let diff = (result - expected).abs();
    assert!(diff < 1e-9, "Expected {expected}, got {result}");
}
```

## Async Tests with #[tokio::test]

```rust
use tokio::time::{sleep, Duration};

#[tokio::test]
async fn async_service_returns_users() {
    let service = UserService::new();
    let users = service.list_all().await.unwrap();
    assert!(!users.is_empty());
}

#[tokio::test]
async fn timeout_is_respected() {
    let result = tokio::time::timeout(
        Duration::from_millis(100),
        slow_operation(),
    ).await;
    assert!(result.is_err(), "Expected timeout");
}
```

## Integration Tests in tests/

```rust
// tests/users_api.rs
// Integration tests run as a separate binary — only access public API

mod common;

use common::spawn_test_server;

#[tokio::test]
async fn create_user_returns_201() {
    let server = spawn_test_server().await;
    let client = reqwest::Client::new();

    let response = client
        .post(format!("{}/api/users", server.base_url()))
        .json(&serde_json::json!({ "name": "Alice", "email": "alice@example.com" }))
        .send()
        .await
        .unwrap();

    assert_eq!(response.status(), 201);
    let body: serde_json::Value = response.json().await.unwrap();
    assert_eq!(body["name"], "Alice");
}

#[tokio::test]
async fn get_nonexistent_user_returns_404() {
    let server = spawn_test_server().await;
    let client = reqwest::Client::new();

    let response = client
        .get(format!("{}/api/users/99999", server.base_url()))
        .send()
        .await
        .unwrap();

    assert_eq!(response.status(), 404);
}
```

## Shared Test Helpers (tests/common/mod.rs)

```rust
// tests/common/mod.rs
use std::net::TcpListener;

pub struct TestServer {
    base_url: String,
    _handle: tokio::task::JoinHandle<()>,
}

impl TestServer {
    pub fn base_url(&self) -> &str {
        &self.base_url
    }
}

pub async fn spawn_test_server() -> TestServer {
    let listener = TcpListener::bind("127.0.0.1:0").unwrap();  // OS assigns port
    let port = listener.local_addr().unwrap().port();

    let db = setup_test_db().await;
    let app = build_app(db);

    let handle = tokio::spawn(async move {
        axum::serve(
            tokio::net::TcpListener::from_std(listener).unwrap(),
            app,
        )
        .await
        .unwrap();
    });

    TestServer {
        base_url: format!("http://127.0.0.1:{port}"),
        _handle: handle,
    }
}

async fn setup_test_db() -> sqlx::PgPool {
    let url = std::env::var("TEST_DATABASE_URL")
        .unwrap_or_else(|_| "postgres://postgres:postgres@localhost:5432/test_db".to_string());
    let pool = sqlx::PgPool::connect(&url).await.unwrap();
    sqlx::migrate!("./migrations").run(&pool).await.unwrap();
    pool
}
```

## Mocking with mockall

```rust
// Cargo.toml
// [dev-dependencies]
// mockall = "0.13"

use mockall::{automock, predicate::*};

#[automock]
pub trait UserRepository: Send + Sync {
    async fn find_by_id(&self, id: i64) -> Result<Option<User>, sqlx::Error>;
    async fn save(&self, user: &User) -> Result<User, sqlx::Error>;
}

#[cfg(test)]
mod tests {
    use super::*;
    use mockall::predicate;

    #[tokio::test]
    async fn service_returns_user_when_found() {
        let mut mock_repo = MockUserRepository::new();

        mock_repo
            .expect_find_by_id()
            .with(predicate::eq(42i64))
            .times(1)
            .returning(|_| Ok(Some(User { id: 42, name: "Alice".into(), email: "alice@example.com".into() })));

        let service = UserService::new(Arc::new(mock_repo));
        let result = service.get_user(42).await.unwrap();

        assert_eq!(result.unwrap().name, "Alice");
    }

    #[tokio::test]
    async fn service_returns_none_when_not_found() {
        let mut mock_repo = MockUserRepository::new();

        mock_repo
            .expect_find_by_id()
            .returning(|_| Ok(None));

        let service = UserService::new(Arc::new(mock_repo));
        let result = service.get_user(999).await.unwrap();

        assert!(result.is_none());
    }

    #[tokio::test]
    async fn service_propagates_db_error() {
        let mut mock_repo = MockUserRepository::new();

        mock_repo
            .expect_find_by_id()
            .returning(|_| Err(sqlx::Error::RowNotFound));

        let service = UserService::new(Arc::new(mock_repo));
        let result = service.get_user(1).await;

        assert!(result.is_err());
    }
}
```

## Test Fixtures with tempfile

```rust
// Cargo.toml
// [dev-dependencies]
// tempfile = "3"

use tempfile::TempDir;
use std::path::PathBuf;

fn create_temp_config() -> (TempDir, PathBuf) {
    let dir = TempDir::new().unwrap();  // auto-deleted when TempDir drops
    let config_path = dir.path().join("config.toml");
    std::fs::write(&config_path, r#"
        [server]
        port = 8080
    "#).unwrap();
    (dir, config_path)
}

#[test]
fn config_loader_reads_file() {
    let (_dir, path) = create_temp_config();  // _dir must live until end of test
    let config = Config::from_file(&path).unwrap();
    assert_eq!(config.server.port, 8080);
}
```

## Running Tests

```bash
cargo test                          # run all tests
cargo test user_service             # filter by name
cargo test -- --nocapture           # show println! output
cargo test -- --test-threads=1      # sequential (for DB tests sharing state)
cargo test --lib                    # unit tests only
cargo test --test users_api         # specific integration test file
cargo nextest run                   # cargo-nextest: faster parallel runner
```

## Rules

- Write unit tests in `#[cfg(test)]` modules inside the same file as the code — tests stay close to the implementation.
- Write integration tests in the `tests/` directory — they compile as a separate crate and test only the public API.
- Use `#[automock]` on traits (not structs) to generate mocks — design services around trait interfaces for testability.
- Use `#[tokio::test]` for async tests — standard `#[test]` cannot `await`.
- Use `tempfile::TempDir` for filesystem tests — it auto-cleans on drop.
- Keep `_handle` or similar fields alive for the duration of server-based integration tests (drop == shutdown).
- Avoid `unwrap()` in production code; `unwrap()` in tests is acceptable since panics become clear test failures.
- Use `cargo nextest` for faster test execution in CI — it runs tests in separate processes.
