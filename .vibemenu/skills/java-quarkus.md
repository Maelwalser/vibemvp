# Java + Quarkus Skill Guide

## Project Layout

```
service-name/
├── pom.xml
├── src/main/java/org/example/
│   ├── resource/     # JAX-RS resources (HTTP handlers)
│   ├── service/      # Business logic
│   ├── repository/   # Data access (Panache)
│   ├── model/        # Entities and DTOs
│   └── config/       # ConfigProperty groups
├── src/main/resources/
│   └── application.properties
└── src/test/java/org/example/
```

## pom.xml Boilerplate

```xml
<dependencies>
  <dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-resteasy-reactive-jackson</artifactId>
  </dependency>
  <dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-hibernate-orm-panache</artifactId>
  </dependency>
  <dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-jdbc-postgresql</artifactId>
  </dependency>
  <dependency>
    <groupId>io.quarkus</groupId>
    <artifactId>quarkus-arc</artifactId>
  </dependency>
</dependencies>
```

## Resource (Handler) Pattern

```java
@Path("/users")
@Produces(MediaType.APPLICATION_JSON)
@Consumes(MediaType.APPLICATION_JSON)
public class UserResource {

    @Inject
    UserService userService;

    @GET
    public List<UserDTO> list() {
        return userService.listAll();
    }

    @GET
    @Path("/{id}")
    public Response getById(@PathParam("id") Long id) {
        return userService.findById(id)
            .map(u -> Response.ok(u).build())
            .orElse(Response.status(Response.Status.NOT_FOUND).build());
    }

    @POST
    @Transactional
    public Response create(@Valid CreateUserRequest req) {
        UserDTO created = userService.create(req);
        return Response.status(Response.Status.CREATED).entity(created).build();
    }

    @DELETE
    @Path("/{id}")
    @Transactional
    public Response delete(@PathParam("id") Long id) {
        boolean deleted = userService.delete(id);
        return deleted
            ? Response.noContent().build()
            : Response.status(Response.Status.NOT_FOUND).build();
    }
}
```

## Panache ORM Pattern

```java
@Entity
@Table(name = "users")
public class UserEntity extends PanacheEntity {
    // id inherited from PanacheEntity as Long

    @Column(nullable = false)
    public String name;

    @Column(unique = true, nullable = false)
    public String email;

    // Custom finders as static methods
    public static Optional<UserEntity> findByEmail(String email) {
        return find("email", email).firstResultOptional();
    }

    public static List<UserEntity> findActive() {
        return list("active = true order by name");
    }
}
```

## CDI Dependency Injection

```java
@ApplicationScoped
public class UserService {

    @Inject
    UserRepository userRepository;

    public List<UserDTO> listAll() {
        return userRepository.listAll().stream()
            .map(UserDTO::from)
            .collect(Collectors.toList());
    }

    @Transactional
    public UserDTO create(CreateUserRequest req) {
        UserEntity entity = new UserEntity();
        entity.name = req.name();
        entity.email = req.email();
        userRepository.persist(entity);
        return UserDTO.from(entity);
    }
}

@ApplicationScoped
public class UserRepository implements PanacheRepository<UserEntity> {
    // Inherits: listAll(), findById(), persist(), delete(), count(), etc.
    public Optional<UserEntity> findByEmail(String email) {
        return find("email", email).firstResultOptional();
    }
}
```

## Configuration

```java
@ConfigMapping(prefix = "app")
public interface AppConfig {
    String apiKey();
    DatabaseConfig database();

    interface DatabaseConfig {
        String url();
        int maxPoolSize();
    }
}

// application.properties
// app.api-key=${API_KEY}
// app.database.url=${DATABASE_URL}
// app.database.max-pool-size=10
```

## Reactive Routes (RESTEasy Reactive)

```java
@Path("/stream")
public class StreamResource {

    @GET
    @Produces(MediaType.SERVER_SENT_EVENTS)
    public Multi<String> stream() {
        return Multi.createFrom().ticks().every(Duration.ofSeconds(1))
            .map(t -> "event-" + t);
    }
}
```

## Exception Mapping

```java
@Provider
public class ValidationExceptionMapper implements ExceptionMapper<ConstraintViolationException> {

    @Override
    public Response toResponse(ConstraintViolationException ex) {
        Map<String, String> errors = ex.getConstraintViolations().stream()
            .collect(Collectors.toMap(
                v -> v.getPropertyPath().toString(),
                ConstraintViolation::getMessage
            ));
        return Response.status(Response.Status.BAD_REQUEST)
            .entity(Map.of("errors", errors))
            .build();
    }
}
```

## Native Image Constraints

- Avoid runtime reflection; annotate classes with `@RegisterForReflection` if reflection is required:
  ```java
  @RegisterForReflection
  public class MyDTO { ... }
  ```
- No dynamic class loading — wire dependencies at build time with CDI
- Use `quarkus.native.additional-build-args` for native-specific resources
- Test native builds locally with `./mvnw package -Pnative`

## Dev Mode

```bash
./mvnw quarkus:dev          # hot reload, dev UI at localhost:8080/q/dev
./mvnw quarkus:test         # continuous testing
./mvnw package -Pnative     # build native binary
```

## Health Check

```java
// Add quarkus-smallrye-health dependency
// Endpoints auto-exposed at /q/health/live and /q/health/ready
@Liveness
@ApplicationScoped
public class DatabaseHealthCheck implements HealthCheck {
    @Override
    public HealthCheckResponse call() {
        return HealthCheckResponse.up("database");
    }
}
```

## Rules

- Annotate all JAX-RS resources with `@Path` at class level; use `@GET`/`@POST`/etc. on methods.
- Use `@Transactional` on service methods that write to the database, not on resource methods.
- Prefer `PanacheRepository<T>` over `PanacheEntity` for testability (easier to mock repositories).
- Never use `System.out` — inject `org.jboss.logging.Logger` or use `@Inject Logger log`.
- For native images, register all reflection targets with `@RegisterForReflection`.
- Read all config via `@ConfigMapping` interfaces, never hardcode values.
- Use `@Valid` on request params to trigger Bean Validation automatically.
