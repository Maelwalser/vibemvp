# Testing: Integration Tests with Testcontainers Skill Guide

## Overview

Testcontainers for Java, Go, and Node.js — database, broker, and cache containers with wait strategies, networking, and cleanup.

## Java (JUnit 5)

```java
// pom.xml dependency
// <dependency>
//   <groupId>org.testcontainers</groupId>
//   <artifactId>postgresql</artifactId>
//   <version>1.19.4</version>
//   <scope>test</scope>
// </dependency>

import org.testcontainers.containers.PostgreSQLContainer;
import org.testcontainers.containers.KafkaContainer;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;
import org.testcontainers.utility.DockerImageName;

@Testcontainers
@SpringBootTest
class UserRepositoryIT {

    @Container
    static PostgreSQLContainer<?> postgres = new PostgreSQLContainer<>("postgres:16-alpine")
        .withDatabaseName("testdb")
        .withUsername("test")
        .withPassword("test")
        .withInitScript("schema.sql");  // runs on startup

    @Container
    static GenericContainer<?> redis = new GenericContainer<>(DockerImageName.parse("redis:7-alpine"))
        .withExposedPorts(6379)
        .waitingFor(Wait.forLogMessage(".*Ready to accept connections.*\\n", 1));

    @DynamicPropertySource
    static void configureProperties(DynamicPropertyRegistry registry) {
        registry.add("spring.datasource.url", postgres::getJdbcUrl);
        registry.add("spring.datasource.username", postgres::getUsername);
        registry.add("spring.datasource.password", postgres::getPassword);
        registry.add("spring.redis.host", redis::getHost);
        registry.add("spring.redis.port", () -> redis.getMappedPort(6379));
    }

    @Autowired
    UserRepository userRepository;

    @Test
    void insertAndFindUser() {
        var user = new User(null, "a@b.com", "Alice");
        var saved = userRepository.save(user);

        assertThat(saved.getId()).isNotNull();
        assertThat(userRepository.findByEmail("a@b.com")).isPresent();
    }
}
```

### Kafka Container

```java
@Container
static KafkaContainer kafka = new KafkaContainer(DockerImageName.parse("confluentinc/cp-kafka:7.6.0"));

// Get bootstrap servers
String bootstrapServers = kafka.getBootstrapServers();
```

### MongoDB Container

```java
@Container
static MongoDBContainer mongo = new MongoDBContainer(DockerImageName.parse("mongo:7"))
    .withExposedPorts(27017);

// connection string
String connectionString = mongo.getConnectionString();
```

## Go (testcontainers-go)

```go
import (
    "context"
    "testing"

    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
    "github.com/stretchr/testify/require"
)

func TestUserRepository(t *testing.T) {
    ctx := context.Background()

    pgContainer, err := postgres.RunContainer(ctx,
        testcontainers.WithImage("postgres:16-alpine"),
        postgres.WithDatabase("testdb"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        postgres.WithInitScripts("schema.sql"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(30*time.Second),
        ),
    )
    require.NoError(t, err)
    t.Cleanup(func() { pgContainer.Terminate(ctx) })

    connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
    require.NoError(t, err)

    db, err := sql.Open("pgx", connStr)
    require.NoError(t, err)
    t.Cleanup(func() { db.Close() })

    repo := NewUserRepository(db)

    // Test
    user, err := repo.Insert(ctx, InsertUserInput{Email: "a@b.com", Name: "Alice"})
    require.NoError(t, err)
    require.NotEmpty(t, user.ID)
}
```

### Redis Container (Go)

```go
func newRedisContainer(t *testing.T, ctx context.Context) string {
    t.Helper()
    container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "redis:7-alpine",
            ExposedPorts: []string{"6379/tcp"},
            WaitingFor:   wait.ForLog("Ready to accept connections"),
        },
        Started: true,
    })
    require.NoError(t, err)
    t.Cleanup(func() { container.Terminate(ctx) })

    host, _ := container.Host(ctx)
    port, _ := container.MappedPort(ctx, "6379")
    return fmt.Sprintf("%s:%s", host, port.Port())
}
```

## Node.js (testcontainers)

```typescript
import { PostgreSqlContainer, StartedPostgreSqlContainer } from '@testcontainers/postgresql';
import { RedisContainer } from '@testcontainers/redis';
import { KafkaContainer } from '@testcontainers/kafka';

describe('UserRepository', () => {
  let container: StartedPostgreSqlContainer;
  let db: Pool;

  beforeAll(async () => {
    container = await new PostgreSqlContainer('postgres:16-alpine')
      .withDatabase('testdb')
      .withUsername('test')
      .withPassword('test')
      .start();

    db = new Pool({ connectionString: container.getConnectionUri() });
    await db.query(fs.readFileSync('schema.sql', 'utf8'));
  }, 60_000);

  afterAll(async () => {
    await db.end();
    await container.stop();
  });

  it('inserts and retrieves a user', async () => {
    const repo = new UserRepository(db);
    const user = await repo.insert({ email: 'a@b.com', name: 'Alice' });

    expect(user.id).toBeTruthy();
    expect(await repo.findByEmail('a@b.com')).toMatchObject({ email: 'a@b.com' });
  });
});
```

### Kafka (Node.js)

```typescript
const kafkaContainer = await new KafkaContainer('confluentinc/cp-kafka:7.6.0').start();

const kafka = new Kafka({
  brokers: [kafkaContainer.getBootstrapServers()],
});
```

## Wait Strategies

```java
// Log pattern (with occurrence count)
Wait.forLogMessage(".*database system is ready.*\\n", 2)

// Health check endpoint
Wait.forHttp("/health").forStatusCode(200).withStartupTimeout(Duration.ofSeconds(30))

// Listening port
Wait.forListeningPort()

// Composite: multiple strategies must all pass
Wait.forAll(Wait.forListeningPort(), Wait.forHealthcheck())
```

## Docker Compose Container (Multi-service)

```java
@Testcontainers
class MultiServiceIT {

    @Container
    static DockerComposeContainer<?> compose = new DockerComposeContainer<>(
        new File("src/test/resources/docker-compose.test.yml"))
        .withExposedService("postgres", 5432,
            Wait.forLogMessage(".*database system is ready.*\\n", 2))
        .withExposedService("redis", 6379,
            Wait.forListeningPort())
        .withLocalCompose(true);

    static String postgresHost() {
        return compose.getServiceHost("postgres", 5432);
    }
    static int postgresPort() {
        return compose.getServicePort("postgres", 5432);
    }
}
```

## Container Networking (Inter-container)

```java
// Create shared network for direct container-to-container communication
Network network = Network.newNetwork();

GenericContainer<?> app = new GenericContainer<>("myapp:latest")
    .withNetwork(network)
    .withNetworkAliases("app");

GenericContainer<?> postgres = new PostgreSQLContainer<>("postgres:16-alpine")
    .withNetwork(network)
    .withNetworkAliases("postgres");

// app can connect to "postgres:5432" directly
```

## Key Rules

- Use `@Container static` (Java) — containers are shared across all tests in the class, not recreated per test.
- Always call `t.Cleanup()` (Go) or `afterAll` (Node.js) to terminate containers — never rely on GC.
- Use `withInitScript` / `WithInitScripts` to run migrations inside the container at startup.
- Prefer module-specific containers (`PostgreSqlContainer`, `RedisContainer`) over `GenericContainer` when available.
- Set a startup timeout — default may be too short for Kafka (use 60–120 seconds).
- Parallelize container creation with `beforeAll` / `@BeforeAll static` to reduce test suite time.
- Never hardcode ports — always use `getMappedPort()` / `MappedPort()` to get the host-assigned port.
