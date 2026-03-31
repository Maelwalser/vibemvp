# System Declaration Menu — Specification

> A declarative menu for fully specifying a web system's architecture, data, contracts, frontend, and infrastructure. Each section builds context that downstream sections reference (e.g., Domains defined in **Data** are selectable in **Backend** modules).

---

## Global Concepts

These are referenced throughout the menu:

| Concept          | Definition                                                                 |
|------------------|----------------------------------------------------------------------------|
| **Domain**       | A bounded context with its own entities/attributes (defined in Data tab)   |
| **Service Unit** | A microservice, module, or the monolith itself — any deployable backend unit |
| **Contract**     | A typed interface between any two boundaries (frontend↔backend, service↔service) |
| **Data Flow**    | A directional description of when/why/what data moves between two units    |

---

## 1 · Backend Tab

### 1.1 Architecture Pattern *(top-level selector)*

| Option              | Unlocks                                                        |
|---------------------|----------------------------------------------------------------|
| Monolith            | Single service unit config                                     |
| Modular Monolith    | Module registry + internal communication                       |
| Microservices       | Service registry + inter-service communication + API gateway   |
| Event-Driven        | Service registry + broker config + event catalog               |
| Hybrid              | Mix of the above — tag each service unit with its pattern      |

> Selecting an architecture pattern sets the **communication defaults** and **deployment topology** but every option shares the same service-unit definition shape below.

---

### 1.2 Environment Sub Tab

| Field                  | Options / Input                                                                                              |
|------------------------|--------------------------------------------------------------------------------------------------------------|
| Compute Environment    | `Bare Metal` · `VM` · `Containers (Docker)` · `Kubernetes` · `Serverless (FaaS)` · `PaaS`                   |
| Cloud Provider         | `AWS` · `GCP` · `Azure` · `Cloudflare` · `Hetzner` · `Self-hosted` · `Other (specify)`                      |
| Container Orchestrator | *(if Containers/K8s)* `Docker Compose` · `K3s` · `K8s (managed)` · `Nomad` · `ECS` · `Cloud Run`            |
| Region(s)              | Multi-select or free text                                                                                    |
| Environment Stages     | Checkboxes: `Development` · `Staging` · `Production`  — each can override provider/region                   |

---

### 1.3 Service Units Sub Tab

This is the **shared shape** for defining any backend unit — whether it's the single monolith, a module, or a microservice.

#### Adding / Editing a Service Unit

**Identity**

| Field           | Input                                                                                                  |
|-----------------|--------------------------------------------------------------------------------------------------------|
| Name            | Free text identifier (e.g., `auth-service`, `billing-module`)                                          |
| Responsibility  | Free text description of what this unit owns                                                           |
| Domains Handled | Multi-select from domains created in **Data → Domains** tab                                            |
| Pattern Tag     | *(Hybrid only)* `Monolith part` · `Modular module` · `Microservice` · `Event processor` · `Serverless function` |

**Technology**

| Field     | Options                                                                                                          |
|-----------|------------------------------------------------------------------------------------------------------------------|
| Language  | `Go` · `TypeScript/Node` · `Python` · `Java` · `Kotlin` · `C#/.NET` · `Rust` · `Ruby` · `PHP` · `Elixir` · `Other` |
| Framework | *(dynamically filtered by language selection)*                                                                    |

Framework suggestions per language:

| Language        | Frameworks                                                            |
|-----------------|-----------------------------------------------------------------------|
| Go              | `Fiber` · `Gin` · `Echo` · `Chi` · `net/http (stdlib)` · `Connect`   |
| TypeScript/Node | `Express` · `Fastify` · `NestJS` · `Hono` · `tRPC` · `Elysia (Bun)` |
| Python          | `FastAPI` · `Django` · `Flask` · `Litestar` · `Starlette`            |
| Java            | `Spring Boot` · `Quarkus` · `Micronaut` · `Jakarta EE`               |
| Kotlin          | `Ktor` · `Spring Boot (Kotlin)` · `http4k`                           |
| C#/.NET         | `ASP.NET Core` · `Minimal APIs` · `Carter`                           |
| Rust            | `Axum` · `Actix-web` · `Rocket` · `Warp`                             |
| Ruby            | `Rails` · `Sinatra` · `Hanami` · `Roda`                              |
| PHP             | `Laravel` · `Symfony` · `Slim` · `Laminas`                           |
| Elixir          | `Phoenix` · `Plug` · `Bandit`                                        |

**Exposed Interfaces** *(what this unit offers to others)*

Repeatable — add as many as needed:

| Field          | Options / Input                                                                 |
|----------------|---------------------------------------------------------------------------------|
| Interface Name | Free text (e.g., `REST API`, `gRPC Service`, `Event Producer`)                  |
| Type           | `REST` · `GraphQL` · `gRPC` · `WebSocket` · `Event/Message` · `Internal Call`   |
| Description    | Brief summary of what this interface provides                                   |

---

### 1.4 Communication Sub Tab

> Shared across **all** architecture patterns. Each communication link is a directed edge in the system graph.

#### Adding a Communication Link

| Field         | Input                                                                                                    |
|---------------|----------------------------------------------------------------------------------------------------------|
| From          | Select a service unit                                                                                    |
| To            | Select a service unit                                                                                    |
| Direction     | `Unidirectional (→)` · `Bidirectional (↔)` · `Pub/Sub (fan-out)`                                        |
| Protocol      | `REST (HTTP)` · `gRPC` · `GraphQL` · `WebSocket` · `Message Queue` · `Event Bus` · `Internal (in-process)` |
| Data Contract | Select from **Contracts tab** or define inline                                                           |
| Trigger / Flow | Free text description of **when** this communication happens (e.g., "On user registration", "Every 5 min via cron", "When order.placed event fires") |
| Sync/Async    | `Synchronous` · `Asynchronous` · `Fire-and-forget`                                                      |
| Resilience    | *(optional)* Checkboxes: `Retry` · `Circuit breaker` · `Timeout` · `DLQ (dead letter queue)` · `Idempotent` |

---

### 1.5 Messaging Sub Tab

> Visible when architecture includes event-driven patterns or when any communication link uses `Message Queue` / `Event Bus`.

**Broker Configuration**

| Field                | Options / Input                                                                    |
|----------------------|------------------------------------------------------------------------------------|
| Broker Technology    | `Kafka` · `NATS` · `RabbitMQ` · `Redis Streams` · `AWS SQS/SNS` · `Google Pub/Sub` · `Azure Service Bus` · `Pulsar` |
| Deployment           | `Managed (cloud)` · `Self-hosted` · `Embedded`                                    |
| Serialization Format | `JSON` · `Protobuf` · `Avro` · `MessagePack` · `CloudEvents`                      |
| Delivery Guarantee   | `At-most-once` · `At-least-once` · `Exactly-once`                                 |

**Event Catalog** *(repeatable)*

| Field         | Input                                                           |
|---------------|-----------------------------------------------------------------|
| Event Name    | e.g., `order.placed`, `user.registered`                         |
| Domain        | Select from **Data → Domains**                                  |
| Producer(s)   | Multi-select service units                                      |
| Consumer(s)   | Multi-select service units                                      |
| Payload       | Select a DTO from **Contracts → DTOs** or define inline schema  |
| Description   | When/why this event fires                                       |

---

### 1.6 API Gateway Sub Tab

> *(Optional — auto-suggested for Microservices/Hybrid patterns)*

| Field               | Options / Input                                                              |
|----------------------|-----------------------------------------------------------------------------|
| Gateway Technology   | `Kong` · `Traefik` · `NGINX` · `Envoy` · `AWS API Gateway` · `Cloudflare Workers` · `Custom (specify)` · `None` |
| Routing Strategy     | `Path-based` · `Header-based` · `Domain-based`                              |
| Features             | Checkboxes: `Rate limiting` · `Load balancing` · `Auth passthrough` · `Request transformation` · `CORS handling` · `Caching` |

---

### 1.7 Auth & Identity Sub Tab

| Field                  | Options / Input                                                                               |
|------------------------|-----------------------------------------------------------------------------------------------|
| Auth Strategy          | `JWT (stateless)` · `Session-based` · `OAuth 2.0 / OIDC` · `API Keys` · `mTLS` · `None`     |
| Identity Provider      | `Self-managed` · `Auth0` · `Clerk` · `Supabase Auth` · `Firebase Auth` · `Keycloak` · `AWS Cognito` · `Other` |
| Authorization Model    | `RBAC` · `ABAC` · `ACL` · `ReBAC` · `Policy-based (OPA/Cedar)` · `Custom`                    |
| Token Storage (client) | `HttpOnly cookie` · `Authorization header (Bearer)` · `WebSocket protocol header` · `Other`   |
| MFA Support            | `None` · `TOTP` · `SMS` · `Email` · `Passkeys/WebAuthn`                                       |

---

## 2 · Data Tab

### 2.1 Databases Sub Tab

#### Adding a Database

| Field              | Options / Input                                                                                            |
|--------------------|------------------------------------------------------------------------------------------------------------|
| Name               | Free text identifier (e.g., `primary-postgres`, `cache-redis`)                                             |
| Category           | `Relational` · `Document` · `Key-Value` · `Wide-Column` · `Graph` · `Time-Series` · `Search` · `Vector`   |
| Technology         | *(filtered by category — see below)*                                                                       |
| Purpose            | Free text (e.g., "Primary user data store", "Session cache", "Chat history")                               |
| Owned By           | Select service unit(s) — which services read/write this database                                           |
| Hosting            | `Managed (cloud)` · `Self-hosted` · `Embedded / In-process`                                                |
| High Availability  | `Single instance` · `Primary-replica` · `Multi-primary` · `Cluster`                                       |

Technology options per category:

| Category     | Technologies                                                              |
|--------------|---------------------------------------------------------------------------|
| Relational   | `PostgreSQL` · `MySQL` · `MariaDB` · `SQLite` · `SQL Server` · `CockroachDB` |
| Document     | `MongoDB` · `CouchDB` · `Firestore` · `DynamoDB` · `FerretDB`            |
| Key-Value    | `Redis` · `Valkey` · `Memcached` · `DynamoDB` · `etcd`                   |
| Wide-Column  | `ScyllaDB` · `Cassandra` · `HBase` · `DynamoDB`                          |
| Graph        | `Neo4j` · `ArangoDB` · `DGraph` · `Amazon Neptune`                       |
| Time-Series  | `TimescaleDB` · `InfluxDB` · `QuestDB` · `Prometheus (storage)`          |
| Search       | `Elasticsearch` · `OpenSearch` · `Meilisearch` · `Typesense`             |
| Vector       | `pgvector` · `Pinecone` · `Qdrant` · `Weaviate` · `Milvus` · `ChromaDB` |

**Database-specific options** *(shown conditionally)*:

- **Relational**: Migration tool (`golang-migrate` · `Flyway` · `Alembic` · `Prisma` · `Drizzle` · `TypeORM` · `Other`), ORM/query builder, connection pooling (`PgBouncer` · `built-in` · `none`)
- **Key-Value**: Eviction policy, persistence mode (`RDB` · `AOF` · `none`), cluster mode toggle
- **Document**: Schema validation toggle, index strategy
- **Wide-Column**: Replication factor, consistency level (`ONE` · `QUORUM` · `ALL`)
- **Search**: Analyzer/tokenizer config, index refresh interval

---

### 2.2 Domains Sub Tab

> Domains are the **source of truth** for your system's data model. They are referenced by service units, contracts, and frontend pages.

#### Adding a Domain

| Field       | Input                                                          |
|-------------|----------------------------------------------------------------|
| Name        | e.g., `User`, `Order`, `Product`, `Message`                   |
| Description | What this domain represents in the business context            |
| Database    | Select which database(s) store this domain's data              |

#### Domain Attributes *(repeatable)*

| Field        | Options / Input                                                                                             |
|--------------|-------------------------------------------------------------------------------------------------------------|
| Name         | e.g., `id`, `email`, `created_at`                                                                           |
| Type         | `String` · `Int` · `Float` · `Boolean` · `DateTime` · `UUID` · `Enum(values)` · `JSON/Map` · `Binary` · `Array(type)` · `Ref(Domain)` |
| Constraints  | Checkboxes: `Required` · `Unique` · `Indexed` · `Primary Key` · `Auto-generated` · `Immutable`              |
| Default      | *(optional)* Default value or generation strategy (`uuid_v4`, `now()`, `auto_increment`)                     |
| Sensitive    | Toggle — marks field for encryption/masking/audit considerations                                             |
| Validation   | *(optional)* Free text or pattern (e.g., `email format`, `min:8 max:128`, `regex:...`)                      |

#### Domain Relationships *(repeatable)*

| Field             | Input                                                    |
|-------------------|----------------------------------------------------------|
| Related Domain    | Select from other domains                                |
| Relationship Type | `One-to-One` · `One-to-Many` · `Many-to-Many`           |
| Foreign Key Field | Which attribute holds the reference                      |
| Cascade Behavior  | `Cascade delete` · `Set null` · `Restrict` · `No action` |

---

### 2.3 Caching Sub Tab

| Field            | Options / Input                                                         |
|------------------|-------------------------------------------------------------------------|
| Caching Layer    | `Application-level` · `Dedicated cache (Redis/Valkey)` · `CDN` · `None` |
| Strategy         | `Cache-aside` · `Read-through` · `Write-through` · `Write-behind`      |
| Invalidation     | `TTL-based` · `Event-driven` · `Manual` · `Hybrid`                     |
| Cached Entities  | Multi-select from domains                                               |

---

### 2.4 File / Object Storage Sub Tab

| Field       | Options / Input                                                           |
|-------------|---------------------------------------------------------------------------|
| Technology  | `S3` · `GCS` · `Azure Blob` · `MinIO` · `Cloudflare R2` · `Local disk`   |
| Purpose     | Free text (e.g., "User avatars", "Document uploads", "Backups")           |
| Access      | `Public (CDN-fronted)` · `Private (signed URLs)` · `Internal only`        |
| Max Size    | *(optional)* Per-file size limit                                          |

---

## 3 · Contracts Tab

> Contracts define the **typed interfaces** between boundaries. They are referenced by communication links, frontend pages, and event payloads.

### 3.1 DTOs Sub Tab

#### Adding a DTO

| Field        | Input                                                                                     |
|--------------|-------------------------------------------------------------------------------------------|
| Name         | e.g., `CreateUserRequest`, `OrderSummaryResponse`, `UserRegisteredEvent`                  |
| Category     | `Request` · `Response` · `Event Payload` · `Shared/Common`                                |
| Source Domain(s) | Multi-select from **Data → Domains** — which domains does this DTO derive from        |
| Description  | What this DTO represents and when it's used                                               |

#### DTO Fields *(repeatable)*

| Field      | Options / Input                                                                                            |
|------------|------------------------------------------------------------------------------------------------------------|
| Name       | e.g., `email`, `order_items`, `total`                                                                      |
| Type       | `string` · `int` · `float` · `boolean` · `datetime` · `uuid` · `enum(values)` · `array(type)` · `nested(DTO)` · `map(key,value)` |
| Required   | Toggle                                                                                                     |
| Nullable   | Toggle                                                                                                     |
| Validation | *(optional)* e.g., `min_length:1`, `max:100`, `email`, `regex:...`                                        |
| Notes      | *(optional)* Mapping notes, transformation hints                                                           |

---

### 3.2 Endpoints / Operations Sub Tab

#### Adding an Endpoint

| Field            | Options / Input                                                       |
|------------------|-----------------------------------------------------------------------|
| Service Unit     | Select which backend unit exposes this                                |
| Name / Path      | e.g., `POST /api/v1/users`, `getUser`, `UserService.Create`          |
| Protocol         | `REST` · `GraphQL` · `gRPC` · `WebSocket message` · `Event`          |
| Auth Required    | Toggle + select auth scope/role if applicable                         |
| Request DTO      | Select from DTOs tab (or `None` for parameter-only)                   |
| Response DTO     | Select from DTOs tab                                                  |
| Error Responses  | Repeatable: status code / error type + description                    |
| Rate Limit       | *(optional)* e.g., `100 req/min per user`                            |
| Description      | What this endpoint does, when the frontend calls it                   |

**REST-specific fields** *(shown conditionally)*:

| Field          | Input                                               |
|----------------|-----------------------------------------------------|
| HTTP Method    | `GET` · `POST` · `PUT` · `PATCH` · `DELETE`         |
| Path Params    | Repeatable: name + type                              |
| Query Params   | Repeatable: name + type + required toggle            |
| Pagination     | `None` · `Offset-based` · `Cursor-based` · `Keyset` |

**GraphQL-specific fields**:

| Field          | Input                                   |
|----------------|-----------------------------------------|
| Operation Type | `Query` · `Mutation` · `Subscription`   |
| Type Defs      | Free text or reference DTOs             |

**gRPC-specific fields**:

| Field         | Input                                            |
|---------------|--------------------------------------------------|
| Service Name  | e.g., `UserService`                              |
| RPC Method    | e.g., `CreateUser`                               |
| Stream Type   | `Unary` · `Server stream` · `Client stream` · `Bidirectional` |

**WebSocket-specific fields**:

| Field          | Input                                              |
|----------------|----------------------------------------------------|
| Channel / Room | e.g., `/ws/chat/{roomId}`                          |
| Client Events  | Repeatable: event name + payload DTO               |
| Server Events  | Repeatable: event name + payload DTO               |

---

### 3.3 API Versioning Sub Tab

| Field              | Options / Input                                                  |
|--------------------|------------------------------------------------------------------|
| Versioning Strategy | `URL path (/v1/)` · `Header (Accept-Version)` · `Query param` · `None` |
| Current Version     | e.g., `v1`                                                      |
| Deprecation Policy  | Free text or `None`                                              |

---

## 4 · Frontend Tab

### 4.1 Technologies Sub Tab

| Field          | Options                                                                                                         |
|----------------|-----------------------------------------------------------------------------------------------------------------|
| Language       | `TypeScript` · `JavaScript` · `Dart` · `Kotlin` · `Swift`                                                      |
| Platform       | `Web (SPA)` · `Web (SSR/SSG)` · `Mobile (cross-platform)` · `Mobile (native)` · `Desktop`                      |
| Framework      | *(filtered by language + platform — see below)*                                                                 |
| Meta-framework | *(if applicable)* e.g., `Next.js`, `Nuxt`, `SvelteKit`, `Remix`, `Astro`                                       |
| Package Manager| `npm` · `yarn` · `pnpm` · `bun`                                                                                |

Framework options per language/platform:

| Language/Platform    | Frameworks                                                       |
|----------------------|------------------------------------------------------------------|
| TypeScript/JS — Web  | `React` · `Vue` · `Svelte` · `Angular` · `Solid` · `Qwik` · `HTMX` |
| TypeScript/JS — SSR  | `Next.js` · `Nuxt` · `SvelteKit` · `Remix` · `Astro`           |
| Dart — Cross-plat    | `Flutter`                                                        |
| Kotlin — Native      | `Jetpack Compose` · `KMP (Compose Multiplatform)`                |
| Swift — Native       | `SwiftUI` · `UIKit`                                              |
| Cross-platform       | `React Native` · `Expo` · `Flutter` · `Tauri` · `Electron`      |

**Additional Tooling**

| Field            | Options                                                                          |
|------------------|----------------------------------------------------------------------------------|
| Styling          | `Tailwind CSS` · `CSS Modules` · `Styled Components` · `Sass/SCSS` · `Vanilla CSS` · `UnoCSS` |
| Component Library| `shadcn/ui` · `Radix` · `Material UI` · `Ant Design` · `Headless UI` · `DaisyUI` · `None` · `Custom` |
| State Management | `React Context` · `Zustand` · `Redux Toolkit` · `Jotai` · `Pinia` · `Svelte stores` · `Signals` · `None` |
| Data Fetching    | `TanStack Query` · `SWR` · `Apollo Client` · `tRPC client` · `RTK Query` · `Native fetch` |
| Form Handling    | `React Hook Form` · `Formik` · `Zod + native` · `Vee-Validate` · `None`         |
| Validation       | `Zod` · `Yup` · `Valibot` · `Joi` · `Class-validator` · `None`                  |

---

### 4.2 Theming Sub Tab

| Field           | Input                                                                   |
|-----------------|-------------------------------------------------------------------------|
| Color Palette   | Primary, secondary, accent, background, surface, error, success, warning — hex/hsl pickers |
| Dark Mode       | `None` · `Toggle (user preference)` · `System preference` · `Dark only` |
| Font — Headings | Font family + weight + source (`Google Fonts` · `Local` · `System`)     |
| Font — Body     | Font family + weight + source                                           |
| Font — Mono     | Font family (for code blocks / technical content)                       |
| Border Radius   | `Sharp (0)` · `Subtle (4px)` · `Rounded (8px)` · `Pill (999px)` · `Custom` |
| Spacing Scale   | `Compact (4px base)` · `Default (8px base)` · `Spacious (12px base)`   |
| Elevation Style | `Shadows` · `Borders` · `Both` · `Flat`                                |
| Motion          | `None` · `Subtle transitions` · `Animated (spring/ease)`               |

---

### 4.3 Pages Sub Tab

#### Adding a Page

**Identity**

| Field        | Input                                                                      |
|--------------|----------------------------------------------------------------------------|
| Name         | e.g., `Dashboard`, `User Profile`, `Checkout`                             |
| Route        | e.g., `/dashboard`, `/users/:id`, `/checkout`                             |
| Auth Required| Toggle + required role(s) if applicable                                    |
| Layout       | `Default` · `Sidebar` · `Full-width` · `Blank` · `Custom (specify)`       |
| Description  | Free text — what this page does, its purpose                               |

**Functionality**

| Field                 | Input                                                         |
|-----------------------|---------------------------------------------------------------|
| Core Actions          | Free text list of what the user can do on this page           |
| API Endpoints Used    | Multi-select from **Contracts → Endpoints**                   |
| Real-time Data        | Toggle — does this page use WebSocket/SSE subscriptions?      |
| Subscriptions         | *(if real-time)* Select WebSocket events from Contracts       |
| Data Models (Local)   | Select DTOs this page works with from **Contracts → DTOs**    |

**Style & Layout**

| Field             | Input                                                              |
|-------------------|--------------------------------------------------------------------|
| Style Description | Free text describing the visual feel and layout of the page        |
| Key Components    | Free text list of major UI components (e.g., "data table with filters", "profile card with avatar", "multi-step form wizard") |
| Responsive Notes  | *(optional)* Mobile-specific layout or behavior changes            |
| Loading Strategy  | `Skeleton` · `Spinner` · `Progressive` · `Instant (SSR/SSG)`      |
| Error Handling    | `Inline` · `Toast` · `Error boundary / fallback page` · `Retry`   |

---

### 4.4 Navigation Sub Tab

| Field              | Options / Input                                                      |
|--------------------|----------------------------------------------------------------------|
| Nav Type           | `Top bar` · `Sidebar` · `Bottom tabs (mobile)` · `Hamburger menu` · `Combined` |
| Navigation Items   | Repeatable: label + target page + icon + visibility condition        |
| Breadcrumbs        | Toggle                                                               |
| Auth-Aware         | Toggle — show/hide items based on auth state                         |

---

## 5 · Infrastructure Tab *(new)*

### 5.1 Networking Sub Tab

| Field            | Options / Input                                                       |
|------------------|-----------------------------------------------------------------------|
| DNS Provider     | `Cloudflare` · `Route53` · `Cloud DNS` · `Other`                     |
| TLS/SSL          | `Let's Encrypt` · `Cloudflare` · `ACM` · `Manual` · `None (dev)`     |
| Reverse Proxy    | `Nginx` · `Caddy` · `Traefik` · `Cloudflare Tunnel` · `Cloud LB`    |
| CDN              | `Cloudflare` · `CloudFront` · `Fastly` · `Vercel Edge` · `None`      |

### 5.2 CI/CD Sub Tab

| Field            | Options / Input                                                        |
|------------------|------------------------------------------------------------------------|
| CI/CD Platform   | `GitHub Actions` · `GitLab CI` · `Jenkins` · `CircleCI` · `ArgoCD` · `Tekton` |
| Container Registry | `Docker Hub` · `GHCR` · `ECR` · `GCR` · `Self-hosted`              |
| Deployment Strategy | `Rolling` · `Blue-green` · `Canary` · `Recreate`                  |
| IaC Tool         | `Terraform` · `Pulumi` · `CloudFormation` · `Ansible` · `None`        |

### 5.3 Observability Sub Tab

| Field        | Options / Input                                                                    |
|--------------|------------------------------------------------------------------------------------|
| Logging      | `Loki + Grafana` · `ELK Stack` · `CloudWatch` · `Datadog` · `Stdout/file`         |
| Metrics      | `Prometheus + Grafana` · `Datadog` · `CloudWatch` · `New Relic` · `None`           |
| Tracing      | `OpenTelemetry + Jaeger` · `OpenTelemetry + Tempo` · `Datadog APM` · `None`        |
| Error Tracking | `Sentry` · `Datadog` · `Rollbar` · `Built-in` · `None`                          |
| Health Checks | Toggle — auto-generate `/health` and `/ready` endpoints per service unit          |
| Alerting     | `Grafana Alerting` · `PagerDuty` · `OpsGenie` · `CloudWatch Alarms` · `None`      |

---

## 6 · Cross-Cutting Concerns Tab *(new)*

### 6.1 Testing Strategy Sub Tab

| Field             | Options / Input                                                                  |
|-------------------|----------------------------------------------------------------------------------|
| Unit Testing      | `Jest` · `Vitest` · `pytest` · `Go testing` · `JUnit` · `xUnit` · `Other`       |
| Integration Tests | `Testcontainers` · `Docker Compose` · `In-memory fakes` · `None`                |
| E2E Testing       | `Playwright` · `Cypress` · `Selenium` · `None`                                  |
| API Testing       | `Bruno` · `Hurl` · `Postman/Newman` · `REST Client` · `None`                    |
| Load Testing      | `k6` · `Locust` · `Artillery` · `JMeter` · `None`                               |
| Contract Testing  | `Pact` · `Schemathesis` · `Dredd` · `None`                                      |

### 6.2 Documentation Sub Tab

| Field           | Options / Input                                                        |
|-----------------|------------------------------------------------------------------------|
| API Docs        | `OpenAPI/Swagger` · `GraphQL Playground` · `gRPC reflection` · `None`  |
| Auto-generation | Toggle — generate specs from code annotations                          |
| Changelog       | `Conventional Commits` · `Manual` · `None`                             |

---

## Dependency & Reference Graph

The tabs form a directed dependency graph — this is the intended order of definition:

```
Data (Domains, Databases)
    ↓
Backend (Service Units reference Domains)
    ↓
Contracts (DTOs reference Domains, Endpoints reference Service Units)
    ↓
Frontend (Pages reference Endpoints + DTOs)
    ↓
Infrastructure (references all deployable units)
    ↓
Cross-Cutting (references everything)
```

> **However**, the UI should allow **non-linear editing** — users can start anywhere and link entities later. Empty references show as "unlinked" placeholders that can be resolved.
