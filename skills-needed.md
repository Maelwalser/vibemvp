# Comprehensive Skill Matrix for System Declaration

> Each entry maps to a markdown skill file that an AI code-generation agent loads at runtime.
> File names use kebab-case and live in `.vibemvp/skills/`.

---

## 1. Architectural Patterns

The fundamental structure of the backend dictates component boundaries, routing, and deployment constraints.

* `arch-monolith.md` (Single deployable unit, layered architecture, shared DB, modular package structure)
* `arch-modular-monolith.md` (In-process module boundaries, module registries, internal API contracts, dependency inversion between modules)
* `arch-microservices.md` (Service discovery, circuit breakers, distributed tracing, inter-service auth)
* `arch-event-driven.md` (Event choreography, idempotency, eventual consistency, outbox pattern)
* `arch-serverless.md` (Cold start mitigation, stateless execution, FaaS constraints, per-invocation cost model)
* `arch-hybrid.md` (Mixed pattern tagging, per-service pattern selection, shared infrastructure concerns)

---

## 2. Backend Languages & Frameworks

Explicit routing, middleware, and dependency injection patterns for each supported framework.

### Go (Golang)
* `go-fiber.md` (fasthttp-based routing, middleware with c.Next(), Fiber groups, error wrapping with fiber.NewError(), fasthttp incompatibility with std net/http)
* `go-gin.md` (Routing, middleware chains, context management, gin.H response helpers)
* `go-echo-chi.md` (Echo/Chi routing paradigms, group routing, binder interfaces)
* `go-connectrpc.md` (ConnectRPC implementation and code generation, protobuf integration)
* `go-stdlib-nethttp.md` (Idiomatic standard library implementations, ServeMux, handler interfaces)

### TypeScript / Node.js
* `node-express.md` (Middleware, error handling, router structuring, req/res cycle)
* `node-fastify.md` (Plugin architecture, schema validation with JSON Schema, lifecycle hooks)
* `node-nestjs.md` (Dependency injection, decorators, modules, providers, guards, interceptors)
* `node-hono-elysia.md` (Edge-optimized routing, Bun runtime, typed responses)
* `node-trpc.md` (End-to-end typesafe API definitions, routers, procedures, context)

### Python
* `python-fastapi.md` (Pydantic models, async routes, dependency injection, OpenAPI auto-generation)
* `python-django-drf.md` (Django REST Framework serializers, ViewSets, router configuration, permissions, throttling, N+1 query prevention with select_related/prefetch_related, pagination)
* `python-flask-litestar.md` (Microframework routing, application factory pattern, Litestar DTOs)
* `python-starlette.md` (ASGI async-native patterns, lifespan context managers for DB connection pools, middleware ordering, background tasks, streaming responses, WebSockets)

### Java & Kotlin
* `java-spring-boot.md` (REST controllers, Spring Data JPA, Spring Security, method-level authorization with @PreAuthorize/@PostAuthorize, lazy loading pitfalls, transactional boundaries, @Query custom SQL)
* `java-quarkus.md` (Reactive routes, GraalVM native image constraints, Panache ORM)
* `java-micronaut.md` (Compile-time DI, AOP, GraalVM support)
* `java-jakarta-ee.md` (Enterprise specifications, CDI, JAX-RS)
* `kotlin-ktor.md` (Routing DSL, plugins/features, kotlinx.serialization, coroutine scope management, Koin DI, WebSocket with Flow)
* `kotlin-spring-boot.md` (Spring Boot with Kotlin idioms, data classes as DTOs, all-open compiler plugin, lateinit vs nullable injection, coroutines with WebFlux, extension functions on Spring APIs)
* `kotlin-http4k.md` (Server as a function paradigms, lenses, filters)

### C# / .NET
* `csharp-aspnetcore.md` (Controller-based APIs, middleware pipeline, attribute routing)
* `csharp-minimal-apis.md` (Delegate-based routing, endpoint filters, typed results)
* `csharp-carter.md` (Carter module routing, ICarterModule)
* `csharp-ef-core.md` (Entity Framework contexts, migrations, LINQ queries, relationships)
* `csharp-xunit.md` (xUnit, NSubstitute, Testcontainers integration)

### Rust
* `rust-axum.md` (Tokio-based async routing, state extractors, tower middleware)
* `rust-actix-web.md` (Actor models, multithreading constraints, data extractors)
* `rust-rocket-warp.md` (Macro-based routing, filter chains, fairings)
* `rust-cargo-testing.md` (Unit testing, mocking traits, integration test layout)

### Ruby, PHP & Elixir
* `ruby-rails-sinatra.md` (ActiveRecord, MVC, routing, concerns)
* `ruby-hanami.md` (Repository pattern with Hanami::Repository, action objects with .call() interface, dry-types validation, explicit view exposure, immutable struct entities)
* `ruby-roda.md` (Tree-based routing DSL, plugin architecture, r.remaining_path, r.halt patterns, first-match semantics)
* `ruby-rspec.md` (Behavior-driven development, shared examples, let/subject)
* `php-laravel-symfony.md` (Eloquent ORM, service containers, routing, Symfony DI)
* `php-slim.md` (PSR-7 immutable request/response, PSR-15 middleware, container-based DI with PHP-DI, route groups, ErrorMiddleware)
* `php-laminas.md` (Module config pattern, AbstractController, Service Manager factories, InputFilter validation, View Model patterns)
* `php-phpunit.md` (Testing conventions, mocking, data providers)
* `elixir-phoenix.md` (OTP, GenServer, Channels, LiveView)
* `elixir-plug-bandit.md` (Connection transformations, Plug pipeline, Bandit web server)

---

## 3. Communication Protocols & Gateways

Network-level interface definitions and API gateway configurations.

* `protocol-graphql.md` (Schema definition, resolvers, Dataloaders, subscriptions)
* `protocol-grpc.md` (Protobuf schemas, streaming RPCs, interceptors, error codes)
* `protocol-websockets.md` (Connection state management, broadcasting, heartbeat, reconnection)
* `protocol-sse.md` (Server-Sent Events endpoint implementation per framework, EventSource client, reconnection with last-event-id, event stream format, text/event-stream content type)
* `protocol-mqtt.md` (MQTT broker setup with Mosquitto/EMQX, QoS levels 0/1/2, topic hierarchy design, retained messages, MQTT over WebSocket for browsers, last will and testament, IoT device patterns)
* `gateway-kong-traefik.md` (Declarative routing, plugin configs, middleware chains)
* `gateway-nginx-envoy.md` (Reverse proxying, load balancing, upstream health checks)
* `gateway-aws-apigw.md` (AWS API Gateway REST/HTTP/WebSocket APIs, Lambda integration, authorizers, usage plans)
* `gateway-cloudflare-workers.md` (Edge routing with Workers, KV store access, D1 integration, wrangler.toml)
* `api-resilience.md` (Circuit breaker with half-open state, retry with exponential backoff and jitter, timeout configuration, bulkhead isolation thread pool pattern, fallback values, resilience4j/polly/go-resilience patterns per language)
* `api-error-formats.md` (RFC 7807 Problem Details JSON: type/title/status/detail/instance fields, per-language implementation, custom JSON envelope patterns: code/message/data, error serialization middleware)
* `service-discovery.md` (DNS-based service resolution, Consul service registry with health checks, Kubernetes DNS ClusterIP and headless services, Eureka client registration, static config patterns, sidecar proxy service mesh)
* `cors-config.md` (CORS strategy permissive vs strict allowlist vs same-origin, nginx/caddy/traefik CORS header blocks, per-framework middleware Express/FastAPI/Go/Spring, WAF/CDN CORS enforcement, preflight caching with max-age)
* `linting-tools.md` (golangci-lint config with enabled linters, Ruff Python linter and formatter, ESLint flat config, Checkstyle Java rules, ktlint Kotlin formatting, Clippy Rust lints, RuboCop Ruby style, PHP-CS-Fixer rules)

---

## 4. Messaging & Brokers

Broker-specific consumer and producer configurations.

* `broker-kafka.md` (Consumer groups, partitions, offset tracking, exactly-once semantics, Kafka Streams)
* `broker-nats.md` (Core pub/sub, JetStream persistence, queue groups, key-value store)
* `broker-rabbitmq.md` (Exchanges, queues, routing keys, dead letter exchanges, prefetch)
* `broker-redis-streams.md` (Consumer groups, XADD/XREAD, XACK, pending entry lists)
* `broker-cloud.md` (AWS SQS/SNS, GCP Pub/Sub, Azure Service Bus — managed broker patterns)
* `broker-pulsar.md` (Multi-tenancy, tiered storage, geo-replication, Functions compute, subscription types)

---

## 5. Background Job Queues

Asynchronous and scheduled task processing patterns.

* `jobs-temporal.md` (Temporal workflow orchestration: deterministic workflow functions, activity definitions for non-deterministic ops, worker configuration with task queues, signals for long-lived control, queries for state inspection, saga pattern with compensating activities in reverse order on failure, retry policies with setInitialInterval/setMaximumInterval/setBackoffCoefficient, StartToCloseTimeout vs ScheduleToCloseTimeout, Worker Versioning, Temporal Cloud vs self-hosted)
* `jobs-bullmq.md` (BullMQ Node.js Redis-backed queue: Queue/Worker/QueueEvents classes, job options attempts/backoff/priority/delay/removeOnComplete, FlowProducer for DAG-style job dependencies, repeatable jobs with cron patterns and fixed intervals, rate limiting with limiter config, Worker concurrency, dead letter queue patterns, OpenTelemetry integration)
* `jobs-sidekiq-celery.md` (Sidekiq Ruby: Worker include pattern, sidekiq_options queue/retry, perform_async with JSON-serializable args only, sidekiq-cron for scheduling, server/client middleware chain, exponential backoff sidekiq_retry_in block, DeadSet inspection; Celery Python: @app.task decorator, bind=True for self.retry(), beat scheduler with crontab, chain/group/chord canvas primitives for workflow composition, rate_limit per task, ETA/countdown scheduling)
* `jobs-faktory-asynq-river.md` (Faktory language-agnostic: job server setup, worker clients for Go/Ruby/Node, JSON job payload with jid/jobtype/args, middleware chain, 25-retry exponential backoff, Dead Set; Asynq Go+Redis: task definition with type constants, NewClient enqueue with MaxRetry/ProcessIn/Queue options, NewServer with queue weights, ServeMux handlers, SkipRetry sentinel, Scheduler for recurring tasks, asynqmon web UI; River Go+PostgreSQL: JobArgs struct with Kind() method, Worker[T] interface, InsertTx for transactional safety, advisory locks for job uniqueness, QueueConfig with MaxWorkers, discarded state inspection)

---

## 6. Authentication & Authorization

Security constraints, identity management, and access control.

* `auth-jwt-stateless.md` (Signing algorithms RS256/HS256, verification, HttpOnly cookies, refresh token rotation strategies: rotating/non-rotating/sliding window, token blacklisting with Redis, access+refresh token pair patterns)
* `auth-session-based.md` (Server-side session store with Redis/DB, session ID generation with crypto-secure random, HttpOnly/Secure/SameSite cookie attributes, CSRF protection with double-submit cookie or synchronizer token, framework middleware: Express-session, Django sessions, Rails ActionDispatch, Spring Session)
* `auth-apikey.md` (Crypto-secure key generation, hashed key storage with bcrypt/SHA-256, header vs query param injection, key scopes/permissions, rate limiting per key, rotation and revocation patterns)
* `auth-oauth2-oidc.md` (Authorization code flows, PKCE, token introspection, refresh flows, client credentials for M2M)
* `auth-mtls.md` (Certificate validation, client certificate provisioning, mutual TLS setup, certificate rotation)
* `idp-integrations.md` (Auth0, Clerk, Supabase Auth, Firebase Auth, Keycloak, AWS Cognito — SDK initialization, callback handling, user sync, role mapping)
* `authz-rbac-abac.md` (Role and attribute-based access checks, permission middleware, policy evaluation, predefined roles: admin/superadmin/user/moderator/editor/viewer/manager/auditor/owner)
* `authz-rebac.md` (Relationship-based access control, subject/relation/object tuples, SpiceDB/Zanzibar patterns, check API integration, hierarchical team/document ownership)
* `authz-opa-cedar.md` (External policy evaluation, Rego/Cedar policy authoring, OPA sidecar deployment)
* `auth-mfa.md` (TOTP with QR code generation and time-window tolerance using pyotp/speakeasy, backup codes, SMS OTP via Twilio/SNS with rate limiting and brute-force protection, Email OTP with resend logic, WebAuthn/Passkeys registration and authentication ceremony, credential storage of public keys, conditional UI for password managers)
* `web-auth-flows.md` (Redirect OAuth/OIDC flow with PKCE, modal login component pattern, magic link token generation and email delivery, passwordless auth via WebAuthn, social-only login aggregation, frontend token storage in HttpOnly cookie vs memory, silent refresh pattern)

---

## 7. Application Security

WAF, bot protection, rate limiting, and DDoS mitigation.

* `security-waf.md` (Cloudflare WAF custom rule expression language, OWASP managed ruleset with sensitivity tuning, AWS WAF WebACL with managed rule groups and IP sets and ALB/CloudFront/API GW integration, ModSecurity/NGINX ModSec with CRS paranoia levels PL1-PL4, rule exclusion packages for false positive tuning, audit log monitoring)
* `security-captcha.md` (reCAPTCHA v2 checkbox and invisible with site/secret key and server verification via POST to siteverify, reCAPTCHA v3 score-based with 0.0-1.0 threshold and adaptive blocking, hCaptcha widget and server validation at api.hcaptcha.com/siteverify, Cloudflare Turnstile managed/non-interactive/invisible modes with CF siteverify validation, token expiry and single-use enforcement)
* `security-rate-limiting.md` (Token bucket with Redis HASH and Lua scripts for atomic refill, sliding window with Redis ZSET ZREMRANGEBYSCORE/ZCARD/ZADD atomic via EVAL, fixed window with Redis INCR+TTL, leaky bucket with queue drain, per-user/per-IP/per-API-key key strategies, HTTP 429 with Retry-After header, framework integrations: express-rate-limit, @fastify/rate-limit, bucket4j Spring Boot, Go ratelimit middleware, fastapi-limiter)
* `security-bot-ddos.md` (Cloudflare Bot Management bot score thresholds with JS/managed challenge, DataDome behavioral ML with <2ms evaluation, Imperva Hi-Def fingerprinting, CDN-level DDoS via Cloudflare Under Attack mode and L7 managed rules, AWS Shield Standard L3/L4 plus Shield Advanced L7 with ML mitigation, GCP Cloud Armor security policies, traffic baseline establishment and anomaly alerting)

---

## 8. Databases, ORMs & Storage

Data modeling constraints per database paradigm.

### Relational Databases
* `db-postgres.md` (Advanced indexing, JSONB, window functions, CTEs, pg extensions)
* `db-mysql-mariadb.md` (Indexing strategies, isolation levels, JSON columns, replication)
* `db-sqlite.md` (WAL mode, concurrency handling, embedded use cases)
* `db-cockroachdb.md` (Distributed SQL with PostgreSQL wire compatibility, multi-region cluster config, distributed transaction patterns, replication and failover, schema migration with distributed constraints, connection string management across regions)
* `db-sqlserver.md` (T-SQL CTE generation, set-based operation patterns, Gap and Island queries, trigger generation for multi-row awareness, SEQUENCE and IDENTITY, partition functions, encrypted columns)

### Document Databases
* `db-mongodb-couchdb.md` (Document schemas, aggregation pipelines, indexes, change streams)
* `db-ferretdb.md` (MongoDB wire protocol proxy to PostgreSQL, BSON/JSONB mapping, stateless proxy deployment for Kubernetes, SQL pushdown optimization, data migration from MongoDB)
* `db-dynamodb.md` (Single-table design, GSIs, LSIs, DynamoDB streams, capacity modes)
* `db-firestore.md` (Collection/document hierarchy, real-time listeners, security rules, composite indexes)

### Key-Value Stores
* `db-redis-memcached.md` (Caching patterns, TTL, eviction policies, pub/sub, Lua scripting)
* `db-valkey.md` (Redis 7.2 API compatibility as drop-in replacement, multi-threaded I/O configuration, RDMA support, migration patterns from Redis, divergence from Redis 8+ features)
* `db-etcd.md` (Distributed key-value for configuration/service discovery, watch patterns, leases, MVCC)

### Wide-Column
* `db-wide-column.md` (Cassandra/ScyllaDB partition keys, clustering columns, tombstones, consistency levels ONE/QUORUM/ALL/LOCAL_QUORUM/LOCAL_ONE, replication factor)

### OLAP / Columnar
* `db-clickhouse.md` (MergeTree engine family: MergeTree/ReplacingMergeTree/SummingMergeTree/AggregatingMergeTree/VersionedCollapsingMergeTree, PARTITION BY time-based design, ORDER BY for sparse index, materialized views for incremental aggregation, TTL for automatic partition expiry, distributed tables and sharding, SQL extensions PREWHERE for early filtering, compression codecs Delta/DoubleDelta/LZ4/ZSTD, query profiling with query_log)

### Graph Databases
* `db-neo4j-arangodb.md` (Neo4j Cypher queries, node/edge modeling, ArangoDB AQL multi-model queries, graph traversal patterns, edge definition with _from/_to)
* `db-dgraph.md` (GraphQL schema to graph mapping, predicate definition, super-node detection and prevention, ACID transactions, distributed graph traversal planning)
* `db-neptune.md` (Amazon Neptune Gremlin traversal query generation, SPARQL RDF triple patterns, property graph vs RDF model selection, index strategy for vertex/edge properties, bulk import)

### Time-Series
* `db-timeseries.md` (TimescaleDB hypertables, InfluxDB buckets, QuestDB SAMPLE BY and LATEST ON time-bucket aggregation, ASOF/WINDOW/HORIZON JOINs, timezone-aware queries, downsampling/rollup patterns)

### Search Engines
* `db-search.md` (Elasticsearch mappings, tokenizers, aggregations, index templates, OpenSearch API compatibility with ES 7.10.2, security configuration RBAC and field-level, vector search strategy selection)
* `db-meilisearch.md` (Index configuration with searchableAttributes/filterableAttributes, POST-preferred query API, multi-search batch requests, hybrid full-text + semantic search, ranking rules, placeholder search)
* `db-typesense.md` (Collection schema generation, typo tolerance parameter configuration, alphanumeric token handling, faceted search, geo-search, vector field integration)
* `search-algolia.md` (Index configuration with searchableAttributes/attributesForFaceting/customRanking, record structure with objectID, InstantSearch.js widgets for React/Vue/Angular, server-side indexing with saveObjects/partialUpdateObjects, relevance tuning with ranking formula and query rules, synonym management, facet filtering, multi-index federated search)
* `db-postgres-fts.md` (tsvector/tsquery type system, to_tsvector/to_tsquery/plainto_tsquery/websearch_to_tsquery functions, language-specific text search configs, GIN index creation with fastupdate, setweight for column weighting A/B/C/D, ts_rank and ts_rank_cd scoring, ts_headline for context snippets, pg_trgm extension for typo-tolerance with similarity operator)

### Vector Databases
* `db-vector.md` (pgvector indices, similarity search operators, index types IVFFLAT/HNSW)
* `db-weaviate.md` (Vectorizer integration configuration, semantic search with near_text, vector similarity with near_vector, hybrid search with alpha parameter, schema classes and cross-references, batch import, metadata filtering)
* `db-milvus.md` (Deployment mode selection Lite/Standalone/Distributed, collection schema with partitions, ANN index configuration, streaming insert patterns, filtering at scale, horizontal scaling)
* `db-chromadb.md` (PersistentClient vs EphemeralClient, custom embedding function integration, document chunking strategies, RAG pipeline integration with LangChain/LlamaIndex, metadata filtering, update/delete patterns)

### ORMs & Query Builders
* `orm-prisma-drizzle.md` (Schema definitions, migrations, relation queries, type-safe client generation)
* `orm-typeorm-alembic.md` (Entity modeling, decorator-based schema, migration scripts, query builder)
* `orm-flyway.md` (Versioned V{n}__{desc}.sql and Repeatable R__{desc}.sql migration files, flyway_schema_history tracking, SQL vs Java migrations, Spring Boot integration, CI/CD automation, never edit applied migrations)
* `orm-golang-migrate.md` (Migration file pairs .up.sql/.down.sql, advisory locks for multi-instance safety, graceful shutdown handling, CLI vs embedded library, rollback patterns)
* `orm-atlas.md` (Atlas ariga.io: HCL schema definitions with table/index/foreign_key blocks, declarative vs versioned migration modes, diff-based generation via atlas schema diff, schema inspection with atlas schema inspect, atlas schema apply, CI integration with arigaio/atlas-action GitHub Action, multi-tenant migrations, drift detection between environments)
* `orm-liquibase.md` (XML/YAML/SQL/JSON changeset format with author/id attributes, precondition blocks with onFail/onError, context-based execution for environment-specific changes, rollback via rollbackCount/rollbackToDate/rollbackToTag, Spring Boot auto-configuration, never reorder executed changesets, Liquibase Hub integration)
* `orm-pgbouncer.md` (Pooling mode selection session/transaction/statement, configuration file generation, pool size tuning min/max, timeout configuration, centralized vs distributed deployment topology, monitoring and metrics)
* `db-connection-pooling.md` (Min/max pool size, connection timeout, idle timeout, dynamic sizing, separate pools per application component, failover and retry, connection leak detection, keep-alive validation, HikariCP/pgx/node-postgres patterns)

### Object & File Storage
* `storage-s3-gcs.md` (Presigned URLs, multipart uploads, lifecycle policies, bucket policies, CORS, allowed content types enforcement)
* `storage-minio.md` (S3-compatible API, single-node vs distributed mode with erasure coding, bucket lifecycle policies, IAM policy generation, presigned URLs, TLS configuration, performance tuning for NVMe throughput)
* `storage-r2.md` (Cloudflare R2 with zero egress fees, S3-compatible endpoint configuration, Cloudflare Workers binding integration, presigned upload URLs, CORS, custom domain configuration, cache control headers)
* `storage-azure-blob.md` (Container naming by purpose/tenant/retention, user delegation SAS URI generation, lifecycle policy for hot/cool/cold/archive tiers, Smart Tier auto-tiering, blob metadata and tags, redundancy options, encryption)
* `storage-archival.md` (S3 Glacier lifecycle transition rules STANDARD→STANDARD_IA→GLACIER→DEEP_ARCHIVE with day thresholds, GCS Archive class auto-tiering, Azure Archive tier with cool→archive transition, export patterns to parquet/gzip CSV, restore procedures with time-bound recovery, cost estimation per tier)

---

## 9. Data Governance & Compliance

Schema lifecycle, compliance, and data quality patterns.

* `data-governance.md` (Soft-delete with deleted_at column, partial unique indexes WHERE deleted_at IS NULL, view-based filtering, restoration queries; hard-delete cascade planning; archive table strategy with time-based row migration; retention policy enforcement with cron jobs and pg_cron; PII encryption field-level AES-256 with pgcrypto, application-level encryption, transparent view-based decryption; PII masking functions for display; role-based unmasking policies)
* `data-compliance.md` (GDPR right-to-deletion with hard delete + audit log retention, HIPAA immutable audit trail with user/timestamp/purpose, PCI-DSS tokenization replacing card numbers, SOC2 access controls with schema change audit log, data residency geographically partitioned storage, CCPA opt-out handling, PIPEDA consent tracking; compliance mapping data classification levels Public/Internal/Confidential/Restricted)
* `domain-modeling.md` (Bounded context definition, entity/value object/aggregate patterns, domain attributes with constraints required/unique/not_null/min/max/min_length/max_length/email/url/regex/positive/future/past/enum, attribute types String/Int/Float/Boolean/DateTime/UUID/Enum/JSON/Binary/Array/Ref, sensitive field encryption and masking)
* `domain-relationships.md` (One-to-One/One-to-Many/Many-to-Many relationship modeling, foreign key field selection, cascade behaviors CASCADE/SET NULL/SET DEFAULT/RESTRICT/NO ACTION, join table patterns for M2M, relationship-aware DTO generation)
* `domain-caching.md` (Cache-aside pattern, read-through/write-through/write-behind strategies, TTL-based and event-driven invalidation, Redis/Valkey caching layer configuration, CDN cache integration, cached entity selection, TTL presets 30s/1m/5m/15m/1h/24h)
* `domain-dtos.md` (Request/Response/EventPayload/Shared DTO categories, field type mapping from domain to DTO, validation decorator application, nested DTO composition, required/nullable toggles, mapping notes and transformation hints)

---

## 10. External API Integration

Third-party service integration patterns.

* `external-apis.md` (Provider integration pattern: API Key/OAuth2 client credentials/PKCE/Bearer/Basic Auth/mTLS auth mechanisms, base URL and rate limit config, circuit breaker for external calls, retry with exponential backoff and jitter, fallback value strategy, timeout + fail-fast, inbound webhook path definition, webhook signature verification with HMAC-SHA256, idempotency key headers, SDK wrapper pattern vs raw HTTP, error type mapping to internal errors)

---

## 11. Frontend Frameworks & Tooling

UI paradigms, state management, and component patterns.

### Web — SPA Frameworks
* `web-react-spa.md` (Functional components with hooks, React Router v6+ with BrowserRouter/Routes/Route, useParams/useNavigate, protected routes, code splitting with React.lazy/Suspense, useEffect dependency arrays, abort controllers, useMemo/useCallback/React.memo, stable list keys)
* `web-vue-nuxt.md` (Vue 3 Composition API, Pinia, Vue Router, Nuxt 3 file-based routing, composables auto-import, useAsyncData/useFetch, SSR universal rendering, definePageMeta)
* `web-svelte-sveltekit.md` (Svelte reactive declarations with $: labels, two-way bind: directive, transitions, SvelteKit file routing, +page/+layout/+server files, load functions)
* `web-svelte-standalone.md` (Svelte standalone without SvelteKit: writable/readable/derived stores, $-prefix auto-subscription, custom stores, event modifiers, slot patterns, component transitions)
* `web-angular.md` (NgModule declarations, lazy-loaded feature modules, @Component/@Input/@Output decorators, RxJS operators switchMap/mergeMap/catchError, async pipe, BehaviorSubject, takeUntil unsubscribe, reactive forms with FormBuilder, structural directives *ngIf/*ngFor, angular signals: signal()/computed()/effect())
* `web-solid-qwik.md` (Solid.js: createSignal, createMemo, createEffect, createResource for async data, createStore for nested reactivity; Qwik: resumability concept, QRL lazy-loading, onClick$/onChange$ handlers, useClientEffect$, Suspense boundaries)
* `web-htmx.md` (hx-get/post/put/delete, hx-target/hx-swap modes, hx-trigger with polling, server-sent events hx-sse, form submission patterns, hx-push-url for history, HX-Redirect header for server-driven navigation, HTML fragment responses)

### Web — Meta-Frameworks
* `web-nextjs.md` (Next.js App Router file-based routing, layout.tsx/page.tsx/loading.tsx/error.tsx, async server components, 'use client' directive, Server Actions with 'use server', useFormStatus/useFormState, revalidatePath/revalidateTag, API routes in app/api/, NEXT_PUBLIC env vars, Image optimization, ISR)
* `web-remix-astro.md` (Remix: loader/action pattern with LoaderFunctionArgs, useLoaderData, Form for progressive enhancement, useActionData, useFetcher, nested route layouts; Astro: islands architecture with client:* directives, content collections with Zod schemas, getCollection/getEntryBySlug, hybrid rendering modes)

### Mobile
* `mobile-flutter.md` (Dart widget trees, BLoC pattern, StatelessWidget/StatefulWidget, Navigator 2.0)
* `mobile-jetpack-compose.md` (@Composable annotation, recomposition model, remember/mutableStateOf, rememberSaveable, state hoisting, Column/Row/Box/Modifier, LazyColumn, Material 3 theming, NavHost/NavController with typed routes, LaunchedEffect for coroutines, SideEffect)
* `mobile-kmp-compose.md` (Shared composables in commonMain, expect/actual declarations for platform differences, shared ViewModels with StateFlow, androidMain/iosMain/desktopMain source sets, platform-specific navigation UI)
* `mobile-react-native-expo.md` (Native bridging, view rendering, Expo managed vs bare workflow, EAS build)
* `mobile-swiftui.md` (View protocol, @Observable macro, @State/@Binding/@Environment property wrappers, NavigationStack with NavigationPath, type-safe NavigationDestination, #Preview macro with mock data, ViewBuilder, LazyVStack for performance)
* `mobile-uikit.md` (View controller lifecycle viewDidLoad/viewWillAppear, UITableView/UICollectionView delegate+datasource, Auto Layout with NSLayoutConstraint, UIStackView, target-action pattern, modal presentation, push/pop navigation, differences from SwiftUI imperative model)

### Desktop
* `desktop-tauri-electron.md` (Tauri: #[tauri::command] Rust commands, invoke() from frontend, serde serialization, tauri.conf.json, allowlist security, Tauri APIs for fs/shell/dialog; Electron: main/renderer process split, ipcMain.handle/ipcRenderer.invoke, preload scripts with context isolation, BrowserWindow management, electron-builder packaging)

### Styling & UI Components
* `ui-styling.md` (Tailwind CSS utility classes, CSS Modules scoping, Styled Components/Emotion, Sass/SCSS nesting and variables, UnoCSS atomic CSS, Vanilla CSS custom properties)
* `ui-components.md` (shadcn/ui copy-paste components, Material UI sx prop and theming, Ant Design Form/Table patterns, Radix UI unstyled primitives with Dialog/Dropdown/Tooltip/Tabs composition, Headless UI render props and accessibility, DaisyUI component classes btn/card/input with theming and 30+ built-in themes)

### State Management
* `state-management.md` (Zustand store slices, Redux Toolkit createSlice/createAsyncThunk, Jotai atoms, Pinia defineStore for Vue, Svelte writable/readable/derived stores with $-prefix auto-subscription, Angular Signals signal()/computed()/effect() and interop with RxJS toObservable/toSignal)

### Data Fetching
* `data-fetching.md` (TanStack Query useQuery/useMutation, cache invalidation, optimistic updates; Apollo Client query/mutation/subscription hooks; SWR stale-while-revalidate; tRPC client with type-safe hooks; RTK Query createApi with query/mutation endpoints, tag-based cache invalidation, optimistic updates)

### Form Handling & Validation
* `form-validation.md` (React Hook Form register/handleSubmit/formState, Controller for controlled inputs; Formik with initialValues/onSubmit/validationSchema, Field/ErrorMessage components, FieldArray for dynamic fields, useFormikContext; Zod schema definition and parse; Yup chained validators; Valibot lightweight schemas; Class-validator decorator-based DTOs with @IsString/@IsEmail/@ValidateNested, custom @ValidatorConstraint; Vee-Validate for Vue with defineForm, Field v-slot, schema integration with Zod/Yup)

### Real-time & Progressive Web
* `web-realtime.md` (WebSocket client: new WebSocket(), onmessage/onerror/onclose handlers, reconnection with exponential backoff; SSE EventSource: new EventSource(), named event listeners, last-event-id for resumability, polyfill for IE/edge cases; polling with setInterval and abort controller; connection state management; framework-specific hooks useWebSocket/useEventSource)
* `web-pwa.md` (Web App Manifest: name/short_name/icons 192+512px/display standalone/start_url/theme_color, service worker registration in root layout, SW lifecycle install/activate/fetch events, caching strategies: cache-first for assets, network-first for API, stale-while-revalidate for pages, offline fallback page, BackgroundSync for queued actions, Push Notifications with VAPID keys and pushManager.subscribe(), vite-plugin-pwa, next-pwa configuration)
* `web-image-optimization.md` (Next/Image component with fill property and sizes prop for responsive images; Cloudinary SDK transformations: f_auto/q_auto/w_/h_ parameters, upload presets, signed URLs; Imgix URL parameters for on-the-fly transforms; Sharp self-hosted processing: resize/format/quality pipeline; CDN transform patterns with query string transforms)

### Internationalization
* `web-i18n.md` (i18next: namespace setup, useTranslation hook, plural forms with count, interpolation, LanguageDetector and HttpBackend plugins; next-intl: App Router [locale] segment, NextIntlClientProvider, useTranslations in server and client components, generateMetadata locale support; react-i18next: I18nextProvider, Trans component for JSX in translations; LinguiJS: @lingui/macro t`` template literal, Trans component, lingui extract + lingui compile workflow; vue-i18n: createI18n, useI18n composition API, lazy locale loading; Timezone: UTC storage + display conversion with date-fns-tz/Luxon, user preference detection, Temporal API polyfill)

### Accessibility, SEO & Analytics
* `web-a11y-seo.md` (WCAG A/AA/AAA compliance: color contrast 4.5:1 AA / 7:1 AAA, ARIA labels/describedby/live/expanded/controls/roles, keyboard navigation skip links and focus management, focus-visible CSS for keyboard users, modal focus trap, semantic HTML; SEO: react-helmet meta/canonical/OG/Twitter Cards, Next.js Metadata API static object and generateMetadata async, JSON-LD structured data Organization/Product/Article, sitemap.xml generation with next-sitemap, robots.txt disallow rules, SSR/SSG/ISR rendering strategy for SEO, axe-core and jest-axe for automated checks)
* `web-analytics.md` (PostHog: posthog.init, capture events, identify user, feature flag integration, session replay with PII masking; GA4: gtag config and event calls, dataLayer push, Measurement Protocol; Plausible: script include, custom events, GDPR-friendly no-cookie; Mixpanel: init/track/people.set/group analytics; Segment: Analytics.js 2.0 identify/track/page with 200+ destination routing; consent management: delay script loading until consent, localStorage preference, GDPR/CCPA banner pattern)
* `web-rum.md` (Sentry browser: init with dsn/environment/release, captureException, performance transaction tracing, session replay with replaysSessionSampleRate, source map upload, setUser/setTag context; Datadog RUM: DD_RUM.init with applicationId/clientToken, action tracking, error tracking, session replay, custom addUserAction/addError; LogRocket: init, identify, session recording, Sentry/Jira integration; New Relic Browser: browser agent script injection, addPageAction, SPA soft navigation tracking)

### Frontend Testing & Linting
* `test-frontend.md` (Vitest setup with vite config integration, jsdom environment, coverage with v8/istanbul; Jest with jsdom, moduleNameMapper for assets, transform config; Testing Library: getByRole/getByLabelText/getByText queries, userEvent for interactions, renderHook for custom hooks, waitFor async assertions; Storybook: story format CSF3, play functions for interaction tests, args and argTypes, MSW integration for API mocking)
* `web-linting.md` (ESLint flat config eslint.config.js with recommended rules, TypeScript ESLint, Prettier integration; Biome biome.json as all-in-one linter+formatter replacement; oxlint for fast Rust-based linting; Stylelint CSS/SCSS rules; pre-commit hooks with lint-staged; CI enforcement in GitHub Actions)

---

## 12. Infrastructure, Compute & Deployment

Container orchestration, cloud deployment, and infrastructure-as-code patterns.

### Container & Orchestration
* `infra-kubernetes.md` (Deployments, Services, ConfigMaps, Secrets, Ingress, HPA, PVCs, RBAC, NetworkPolicies)
* `infra-k3s.md` (Lightweight Kubernetes, Traefik ingress built-in, local-path-provisioner, K3s-specific HA with embedded etcd, StatefulSets for stateful workloads)
* `infra-ecs-nomad.md` (ECS task definitions with container definitions/environment/secrets/log config, service configuration with ALB targeting and auto-scaling, Fargate vs EC2 launch types, task execution role vs task role IAM; Nomad job specs)
* `infra-docker-compose.md` (Base/override/prod compose files, service health checks with depends_on conditions, resource limits, restart policies, logging drivers, Docker secrets, volume persistence strategies, converting to Kubernetes manifests)
* `infra-serverless.md` (Cold start mitigation with provisioned concurrency and initialization outside handler, stateless execution patterns, AWS Lambda handler/event/context, SAM template generation, Lambda layers, authorizers, DLQ configuration; GCP Cloud Functions Functions Framework; Cloudflare Workers Service Worker API with KV/D1 bindings; Azure Functions bindings)
* `infra-cloud-run.md` (Cloud Run service manifest with CPU/memory/concurrency/timeout, environment variables with Secret Manager, ingress control, VPC connector, service account binding, revision traffic splitting for canary, Terraform generation)
* `infra-paas.md` (Render render.yaml, Railway railway.json, Fly.io fly.toml with health checks and Machines scaling, Heroku Procfile with web/worker/release dyno types, buildpack configuration)
* `containers-runtime.md` (Node Alpine multi-stage build with tini signal handler and non-root user appgroup/appuser; Go scratch static binary with CGO_ENABLED=0 GOOS=linux -ldflags="-s -w" and CA certs copy; Python slim with pip --no-cache-dir and uv; Distroless gcr.io/distroless/* no shell no package manager with nonroot user; security hardening: read-only root filesystem, no-new-privileges security_opt, cap_drop ALL, tmpfs for writable paths)

### CI/CD Pipelines
* `cicd-pipelines.md` (GitHub Actions workflow stages: lint/typecheck/test/build/scan/deploy, matrix builds for multiple versions/OSes, service containers for integration tests, Docker build+push to GHCR/ECR, pull request previews, scheduled jobs; GitLab CI, Jenkins, CircleCI YAML generation)
* `cicd-gitops.md` (ArgoCD Application CRD with source/destination/sync policy, AppProject governance for access control, multi-environment apps with Kustomize/Helm overlays, image updater for automated version bumping, notification and hook configuration)
* `cicd-tekton.md` (Tekton Task definitions with container steps/workspaces/params/results, Pipeline composition with sequencing and parallelization and when-clauses, PipelineRun and EventListener for webhook triggers)
* `iac-terraform-pulumi.md` (State management, module design, provider configuration, workspace separation per environment, import existing resources, destroy safety)
* `iac-cloudformation.md` (CloudFormation JSON/YAML resource declarations, parameters and outputs, conditions for environment-specific resources, nested stacks, change sets, intrinsic functions Ref/GetAtt/Sub/If, custom Lambda-backed resources)
* `iac-ansible.md` (Playbooks, roles, inventory management, vault for secrets, idempotent task design)

### Deployment Strategies
* `deploy-bluegreen.md` (Parallel blue/green environments, load balancer traffic switch, health check verification before cut-over, rollback by switching back, Kubernetes Service selector swap, AWS ECS target group migration, database schema compatibility requirements)
* `deploy-canary.md` (Weighted traffic distribution 5%→10%→50%→100%, metrics-based promotion with error rate and latency thresholds, automatic rollback on threshold violation, Istio VirtualService with weights, Kubernetes Flagger, AWS AppConfig automated canary, feature flags for code-level gradual rollout)

### Networking
* `infra-networking.md` (Nginx/Caddy/Traefik reverse proxy config, Cloudflare Tunnel zero-trust setup with cloudflared daemon, Route53 DNS records with weighted/latency/failover/geolocation routing, Let's Encrypt via cert-manager ClusterIssuer HTTP01/DNS01, AWS ACM certificate provisioning with Terraform, CloudFront distribution with origin behaviors and cache policies and WAF, Fastly CDN with VCL rules, Cloud Load Balancer backend services, domain strategy subdomain-per-service vs path-based, SSL cert auto-renew with certbot/ACME)

### Secrets & Environment Management
* `secrets-management.md` (HashiCorp Vault: KV v2 read/write, dynamic DB credentials with TTL, AppRole auth with role_id/secret_id, Vault Agent sidecar for auto-renewal, transit encryption for application-level secrets; AWS Secrets Manager: GetSecretValueCommand SDK, automatic rotation, CloudFormation dynamic references; GCP Secret Manager: access_secret_version SDK, IAM binding per secret, versioning; GitHub Secrets: ${{ secrets.NAME }} in workflows, environment-scoped secrets)
* `infra-environments.md` (Multi-stage pipeline dev/staging/qa/preview/prod, promotion pipeline Dev→Staging→Prod with approval gates, per-environment secret isolation vs shared base+overrides, preview environment generation via Vercel/GitHub Actions ephemeral envs/Render preview services, DB migration automation on deploy with pre-deploy hook and rollback on failure, DB seeding with upsert-pattern fixture files and factory patterns, environment-specific feature flag targeting)
* `infra-backup-dr.md` (Cross-region replication for RDS/S3/DynamoDB, daily automated snapshots with retention policy, managed provider DR with RTO/RPO definitions, point-in-time recovery configuration, backup verification testing, failover runbook generation)

---

## 13. Observability & Monitoring

Telemetry, monitoring, alerting, and reliability patterns.

* `obs-logging.md` (Loki + Grafana structured log ingestion, ELK Stack with Logstash pipelines, CloudWatch log groups with retention policies and insights queries, Datadog log collection, structured JSON logging patterns with log level/timestamp/trace_id fields)
* `obs-metrics.md` (Prometheus exporters and recording rules, Grafana dashboard JSON generation, Datadog agent metrics, CloudWatch custom metrics publishing and composite alarms, New Relic APM agent setup and custom instrumentation)
* `obs-tracing.md` (OpenTelemetry SDK setup with auto and manual instrumentation, span creation and baggage propagation, OTLP exporter to Jaeger, OTLP exporter to Grafana Tempo with storage on S3, Datadog APM, sampling strategies head-based and tail-based)
* `obs-error-tracking.md` (Sentry SDK initialization with source maps, Rollbar SDK setup with environment separation and custom context, Datadog error tracking integration)
* `obs-healthchecks.md` (Liveness probe endpoint returning 200, readiness probe with dependency checks for DB/cache/external APIs, Kubernetes livenessProbe/readinessProbe/startupProbe configuration, version and uptime metadata endpoints, health check middleware per language/framework, /healthz and /readyz convention)
* `obs-alerting.md` (Grafana Alerting rule configuration with PromQL thresholds and notification policies, PagerDuty Events API webhook integration with severity mapping, OpsGenie alert routing with team and priority configuration, CloudWatch Alarms with SNS actions)
* `slo-sli.md` (Prometheus recording rules for request success rate SLI and latency p99 SLI and availability SLI, 30-day error budget calculation 99.9%=43.2min downtime, fast burn alert 1h window 30x burn rate, slow burn alert 6h window 6x burn rate, Grafana SLO dashboard with uptime gauge and error budget remaining percentage)

---

## 14. Cross-Cutting: Testing

Validation at every boundary.

* `test-unit.md` (Jest/Vitest unit test suites with mocking; pytest fixtures and parametrize; Go testing package table-driven tests; JUnit 5 with @ParameterizedTest; xUnit with Theory/InlineData)
* `test-integration-containers.md` (Testcontainers for Java with JUnit 5 lifecycle, testcontainers-go with Docker API, testcontainers-node async/await; database containers PostgreSQL/MySQL/MongoDB; broker containers Kafka/RabbitMQ/Redis; wait strategies for logs/health/ports; container composition)
* `test-playwright-cypress.md` (Playwright browser automation with page objects, fixtures, visual comparison, trace viewer; Cypress command chaining, intercept patterns, component testing)
* `test-selenium.md` (Selenium WebDriver initialization Chrome/Firefox, page object model, explicit/implicit waits, screenshot on failure, cross-browser grid setup)
* `test-api.md` (Bruno collection and environment files; Hurl HTTP DSL with request sequences/assertions/variable capture/authentication; Postman/Newman collections for CI)
* `test-load.md` (k6 VU scripts with scenarios and thresholds; Locust Python user classes; Artillery scenario definition with load phases and response time assertions; JMeter thread groups with samplers/controllers/assertions/CSV data sets)
* `test-contract.md` (Pact consumer/provider verification with pact-broker; Schemathesis property-based API testing from OpenAPI spec with stateful operation sequences; Dredd API Blueprint testing with hooks for setup/teardown)

---

## 15. Cross-Cutting: API Contracts & Documentation

Interface definition and developer documentation generation.

* `contracts-openapi.md` (OpenAPI 3.x spec generation, path/operation/schema components, error response definitions, language-specific annotations: JSDoc, Python docstrings, Go struct tags, Swagger UI and ReDoc serving, client SDK code generation from spec)
* `contracts-graphql-docs.md` (GraphQL schema SDL, Query/Mutation/Subscription type definitions, Apollo Sandbox setup, schema introspection configuration, GraphQL Playground, schema-first vs code-first generation)
* `contracts-grpc-reflection.md` (Proto service and message definitions, server reflection configuration, grpcui setup for browser-based testing, protobuflint, buf.build toolchain)
* `contracts-versioning.md` (URL path versioning /v1/ routing, Accept-Version header versioning, query parameter versioning, deprecation policy: Sunset header / versioned removal notice / changelog entry, version lifecycle management)
* `contracts-changelog.md` (Conventional Commits format type(scope): message, breaking change footer, commitlint configuration, pre-commit hooks, semantic-release for automated changelog and version bumping)

---

## 16. DevOps Standards & Engineering Culture

Branch strategies, dependency management, feature flags, and reliability targets.

* `feature-flags.md` (LaunchDarkly: LDClient init, variation() with user context, flag targeting rules gradual rollout/user-ID/segment, multivariate flags, streaming vs polling mode; Unleash: UnleashClient init, isEnabled() with strategy types gradualRolloutUserId/IP/custom, Unleash Edge proxy; Flagsmith: init with environmentID, hasFeature/getFeatureValue for remote config, segments; env-var flags: FEATURE_X=true boolean toggle pattern for teams not using external service)
* `devops-standards.md` (GitHub Flow: main + short-lived feature branches + PR merge with branch protection rules required status checks/dismiss stale reviews/require up-to-date; GitFlow: main/develop/feature/release/hotfix model for versioned releases; Trunk-based: max 1-day branches + feature flags for incomplete code; Dependabot .github/dependabot.yml: npm/docker/pip ecosystems with schedule/ignore/auto-merge config; Renovate renovate.json: preset extends, package grouping, major/minor automerge rules, schedule; code review policy: required approvals 1 or 2, CODEOWNERS, conversation resolution requirement; uptime SLO 99.9%/99.95%/99.99% with Prometheus recording rules; latency P99 targets with histogram quantile alerts)
