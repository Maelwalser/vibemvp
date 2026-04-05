# Java + Jakarta EE Skill Guide

## Project Layout

```
service-name/
├── pom.xml
├── src/main/java/org/example/
│   ├── rest/         # JAX-RS resources
│   ├── service/      # @Stateless / @ApplicationScoped EJBs
│   ├── repository/   # JPA-based data access
│   ├── model/        # @Entity and DTO classes
│   └── config/       # ApplicationConfig (JAX-RS activation)
├── src/main/resources/
│   └── META-INF/
│       └── persistence.xml
└── src/test/java/org/example/
```

## pom.xml Boilerplate

```xml
<dependencies>
  <!-- Jakarta EE Full Platform (provided by runtime: WildFly, Payara, GlassFish, TomEE) -->
  <dependency>
    <groupId>jakarta.platform</groupId>
    <artifactId>jakarta.jakartaee-api</artifactId>
    <version>10.0.0</version>
    <scope>provided</scope>
  </dependency>
</dependencies>

<build>
  <plugins>
    <plugin>
      <groupId>org.apache.maven.plugins</groupId>
      <artifactId>maven-war-plugin</artifactId>
      <version>3.4.0</version>
      <configuration>
        <failOnMissingWebXml>false</failOnMissingWebXml>
      </configuration>
    </plugin>
  </plugins>
</build>
```

## JAX-RS Activation

```java
@ApplicationPath("/api")
public class ApplicationConfig extends Application {
    // Empty class activates JAX-RS; no web.xml needed
}
```

## JAX-RS Resource Pattern

```java
@Path("/users")
@Produces(MediaType.APPLICATION_JSON)
@Consumes(MediaType.APPLICATION_JSON)
@RequestScoped
public class UserResource {

    @Inject
    private UserService userService;

    @GET
    public Response list() {
        List<UserDTO> users = userService.findAll();
        return Response.ok(users).build();
    }

    @GET
    @Path("/{id}")
    public Response getById(@PathParam("id") Long id) {
        UserDTO user = userService.findById(id);
        if (user == null) {
            return Response.status(Response.Status.NOT_FOUND).build();
        }
        return Response.ok(user).build();
    }

    @POST
    public Response create(@Valid CreateUserRequest request) {
        UserDTO created = userService.create(request);
        URI location = UriBuilder.fromResource(UserResource.class)
            .path("/{id}")
            .build(created.id());
        return Response.created(location).entity(created).build();
    }

    @DELETE
    @Path("/{id}")
    public Response delete(@PathParam("id") Long id) {
        userService.delete(id);
        return Response.noContent().build();
    }
}
```

## CDI Beans

```java
// Stateless EJB — preferred for transactional business logic
@Stateless
public class UserService {

    @PersistenceContext
    private EntityManager em;

    @Inject
    private UserRepository userRepository;

    public List<UserDTO> findAll() {
        return userRepository.findAll().stream()
            .map(UserDTO::from)
            .collect(Collectors.toList());
    }

    // @Transactional is the default for @Stateless — no annotation needed
    public UserDTO create(CreateUserRequest request) {
        User user = new User(request.name(), request.email());
        em.persist(user);
        return UserDTO.from(user);
    }
}

// @ApplicationScoped CDI bean (not an EJB) — for non-transactional singletons
@ApplicationScoped
public class CacheService {
    private final Map<Long, UserDTO> cache = new ConcurrentHashMap<>();

    public void put(Long id, UserDTO dto) {
        cache.put(id, dto);
    }

    public Optional<UserDTO> get(Long id) {
        return Optional.ofNullable(cache.get(id));
    }
}
```

## JPA Entity

```java
@Entity
@Table(name = "users")
@NamedQuery(name = "User.findByEmail", query = "SELECT u FROM User u WHERE u.email = :email")
@NamedQuery(name = "User.findActive", query = "SELECT u FROM User u WHERE u.active = true ORDER BY u.name")
public class User {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(nullable = false, length = 200)
    @NotNull
    @Size(max = 200)
    private String name;

    @Column(unique = true, nullable = false, length = 320)
    @NotNull
    @Email
    private String email;

    @Column(nullable = false)
    private boolean active = true;

    // Standard getters/setters
    public Long getId() { return id; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getEmail() { return email; }
    public void setEmail(String email) { this.email = email; }
    public boolean isActive() { return active; }
    public void setActive(boolean active) { this.active = active; }
}
```

## Repository Pattern

```java
@Stateless
public class UserRepository {

    @PersistenceContext
    private EntityManager em;

    public List<User> findAll() {
        return em.createNamedQuery("User.findActive", User.class).getResultList();
    }

    public Optional<User> findById(Long id) {
        return Optional.ofNullable(em.find(User.class, id));
    }

    public Optional<User> findByEmail(String email) {
        try {
            return Optional.of(
                em.createNamedQuery("User.findByEmail", User.class)
                    .setParameter("email", email)
                    .getSingleResult()
            );
        } catch (NoResultException e) {
            return Optional.empty();
        }
    }

    public void persist(User user) {
        em.persist(user);
    }

    public void remove(Long id) {
        findById(id).ifPresent(em::remove);
    }
}
```

## persistence.xml (JNDI Datasource)

```xml
<?xml version="1.0" encoding="UTF-8"?>
<persistence version="3.0"
             xmlns="https://jakarta.ee/xml/ns/persistence">
  <persistence-unit name="primary" transaction-type="JTA">
    <jta-data-source>java:jboss/datasources/UsersDS</jta-data-source>
    <properties>
      <property name="jakarta.persistence.schema-generation.database.action" value="none"/>
      <property name="hibernate.dialect" value="org.hibernate.dialect.PostgreSQLDialect"/>
    </properties>
  </persistence-unit>
</persistence>
```

## Bean Validation

```java
public record CreateUserRequest(
    @NotBlank @Size(max = 200) String name,
    @NotNull @Email String email
) {}

// Validation is triggered by @Valid on JAX-RS resource parameters
// Constraint violations automatically produce 400 Bad Request
```

## @Transactional (CDI Transactions)

```java
// Use on CDI beans (not EJBs — EJBs are transactional by default)
@ApplicationScoped
public class ReportService {

    @Inject
    private EntityManager em;

    @Transactional
    public void generateAndSave(Long userId) {
        // Runs in a transaction; rolled back on any RuntimeException
    }

    @Transactional(Transactional.TxType.REQUIRES_NEW)
    public void auditLog(String event) {
        // Always runs in a new transaction, committed independently
    }
}
```

## Exception Mapping

```java
@Provider
public class ConstraintViolationMapper
    implements ExceptionMapper<ConstraintViolationException> {

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

## Rules

- Activate JAX-RS by extending `Application` with `@ApplicationPath` — no `web.xml` needed in Jakarta EE 10.
- Use `@Stateless` EJBs for transactional services — transactions are required by default with automatic rollback on exceptions.
- Use `@ApplicationScoped` CDI beans for non-transactional singletons (caches, registries).
- Use `@PersistenceContext` to inject `EntityManager` — never create it manually.
- Write `@NamedQuery` annotations on entities for named queries — avoids query string duplication.
- Apply `@Valid` on JAX-RS parameters to trigger Bean Validation; wire `ExceptionMapper` for clean error responses.
- JNDI datasources are configured in the application server, not in code — reference them in `persistence.xml`.
- Never call `em.merge()` on detached entities without understanding the implications — prefer loading by ID and updating fields.
