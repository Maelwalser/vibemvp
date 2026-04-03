# C# + xUnit Skill Guide

## Project Layout

```
ServiceName.Tests/
├── ServiceName.Tests.csproj
├── Unit/
│   ├── Services/
│   │   └── UserServiceTests.cs
│   └── Validators/
├── Integration/
│   ├── Fixtures/
│   │   └── DatabaseFixture.cs   # Testcontainers setup
│   └── Repositories/
│       └── UserRepositoryTests.cs
└── Api/
    ├── Fixtures/
    │   └── WebAppFactory.cs     # WebApplicationFactory<T>
    └── Controllers/
        └── UsersControllerTests.cs
```

## Dependencies

```xml
<PackageReference Include="xunit" Version="2.*" />
<PackageReference Include="xunit.runner.visualstudio" Version="2.*" />
<PackageReference Include="NSubstitute" Version="5.*" />
<PackageReference Include="FluentAssertions" Version="6.*" />
<PackageReference Include="Testcontainers.PostgreSql" Version="3.*" />
<PackageReference Include="Microsoft.AspNetCore.Mvc.Testing" Version="9.*" />
```

## Unit Tests: [Fact] and [Theory]

```csharp
// Unit/Services/UserServiceTests.cs
using FluentAssertions;
using NSubstitute;
using Xunit;

public class UserServiceTests
{
    private readonly IUserRepository _repo = Substitute.For<IUserRepository>();
    private readonly UserService _sut;

    public UserServiceTests()
    {
        _sut = new UserService(_repo);
    }

    [Fact]
    public async Task GetByIdAsync_WhenUserExists_ReturnsUser()
    {
        // Arrange
        var expected = new User { Id = 1, Name = "Alice", Email = "alice@example.com" };
        _repo.GetByIdAsync(1, Arg.Any<CancellationToken>()).Returns(expected);

        // Act
        var result = await _sut.GetByIdAsync(1, CancellationToken.None);

        // Assert
        result.Should().NotBeNull();
        result!.Id.Should().Be(1);
        result.Name.Should().Be("Alice");
    }

    [Fact]
    public async Task GetByIdAsync_WhenUserNotFound_ReturnsNull()
    {
        _repo.GetByIdAsync(99, Arg.Any<CancellationToken>()).Returns((User?)null);

        var result = await _sut.GetByIdAsync(99, CancellationToken.None);

        result.Should().BeNull();
    }

    [Theory]
    [InlineData("")]
    [InlineData("  ")]
    [InlineData(null)]
    public async Task CreateAsync_WithInvalidName_ThrowsValidationException(string? name)
    {
        var req = new UserCreateRequest(name!, "valid@email.com");

        var act = async () => await _sut.CreateAsync(req, CancellationToken.None);

        await act.Should().ThrowAsync<ValidationException>()
            .WithMessage("*name*");
    }
}
```

## NSubstitute Mocking

```csharp
// Setup return values
_repo.GetByIdAsync(1, Arg.Any<CancellationToken>()).Returns(user);
_repo.GetAllAsync(Arg.Any<CancellationToken>()).Returns(new List<User> { user });

// Setup exceptions
_repo.CreateAsync(Arg.Any<User>(), Arg.Any<CancellationToken>())
    .ThrowsAsync(new DbException("unique constraint"));

// Verify calls
await _repo.Received(1).CreateAsync(
    Arg.Is<User>(u => u.Email == "alice@example.com"),
    Arg.Any<CancellationToken>()
);
await _repo.DidNotReceive().DeleteAsync(Arg.Any<int>(), Arg.Any<CancellationToken>());
```

## IClassFixture for Shared Setup

```csharp
// Integration/Fixtures/DatabaseFixture.cs
using Testcontainers.PostgreSql;

public class DatabaseFixture : IAsyncLifetime
{
    private readonly PostgreSqlContainer _container = new PostgreSqlBuilder()
        .WithImage("postgres:16-alpine")
        .WithDatabase("testdb")
        .WithUsername("test")
        .WithPassword("test")
        .Build();

    public string ConnectionString { get; private set; } = string.Empty;
    public AppDbContext Db { get; private set; } = null!;

    public async Task InitializeAsync()
    {
        await _container.StartAsync();
        ConnectionString = _container.GetConnectionString();
        var options = new DbContextOptionsBuilder<AppDbContext>()
            .UseNpgsql(ConnectionString)
            .Options;
        Db = new AppDbContext(options);
        await Db.Database.MigrateAsync();
    }

    public async Task DisposeAsync()
    {
        await Db.DisposeAsync();
        await _container.DisposeAsync();
    }
}

// Integration/Repositories/UserRepositoryTests.cs
public class UserRepositoryTests : IClassFixture<DatabaseFixture>
{
    private readonly AppDbContext _db;
    private readonly UserRepository _sut;

    public UserRepositoryTests(DatabaseFixture fixture)
    {
        _db = fixture.Db;
        _sut = new UserRepository(_db);
        // Clear data between tests
        _db.Users.RemoveRange(_db.Users);
        _db.SaveChanges();
    }

    [Fact]
    public async Task CreateAsync_PersistsUser()
    {
        var user = User.Create("Alice", "alice@example.com");
        await _sut.CreateAsync(user, CancellationToken.None);

        var persisted = await _db.Users.FindAsync(user.Id);
        persisted.Should().NotBeNull();
        persisted!.Email.Should().Be("alice@example.com");
    }
}
```

## API Integration Tests (WebApplicationFactory)

```csharp
// Api/Fixtures/WebAppFactory.cs
using Microsoft.AspNetCore.Mvc.Testing;

public class WebAppFactory : WebApplicationFactory<Program>, IAsyncLifetime
{
    private readonly PostgreSqlContainer _db = new PostgreSqlBuilder()
        .WithImage("postgres:16-alpine")
        .Build();

    protected override void ConfigureWebHost(IWebHostBuilder builder)
    {
        builder.ConfigureServices(services =>
        {
            // Replace real DB with test container
            var descriptor = services.Single(d => d.ServiceType == typeof(DbContextOptions<AppDbContext>));
            services.Remove(descriptor);
            services.AddDbContext<AppDbContext>(opts => opts.UseNpgsql(_db.GetConnectionString()));
        });
    }

    public async Task InitializeAsync() => await _db.StartAsync();
    public new async Task DisposeAsync() => await _db.DisposeAsync();
}

// Api/Controllers/UsersControllerTests.cs
public class UsersControllerTests : IClassFixture<WebAppFactory>
{
    private readonly HttpClient _client;

    public UsersControllerTests(WebAppFactory factory)
        => _client = factory.CreateClient();

    [Fact]
    public async Task POST_users_ReturnsCreated()
    {
        var body = new { name = "Alice", email = "alice@example.com" };

        var response = await _client.PostAsJsonAsync("/api/users", body);

        response.StatusCode.Should().Be(HttpStatusCode.Created);
        var user = await response.Content.ReadFromJsonAsync<UserResponse>();
        user!.Email.Should().Be("alice@example.com");
    }

    [Fact]
    public async Task GET_users_id_WhenNotFound_Returns404()
    {
        var response = await _client.GetAsync("/api/users/99999");

        response.StatusCode.Should().Be(HttpStatusCode.NotFound);
    }
}
```

## FluentAssertions Cheat Sheet

```csharp
result.Should().NotBeNull();
result.Should().Be(expected);
result.Should().BeEquivalentTo(expected);   // deep equality
list.Should().HaveCount(3);
list.Should().Contain(x => x.Id == 1);
list.Should().BeInAscendingOrder(x => x.Name);
act.Should().Throw<ArgumentException>().WithMessage("*name*");
await act.Should().ThrowAsync<ValidationException>();
response.StatusCode.Should().Be(HttpStatusCode.Created);
```

## Error Handling in Tests

- Never swallow exceptions in tests — let them propagate to xUnit.
- Use `Should().ThrowAsync<T>()` to assert on expected exceptions.
- Use `IAsyncLifetime` for async setup/teardown instead of constructors for async work.

## Key Rules

- Use `[Fact]` for single-case tests, `[Theory]`/`[InlineData]` for parameterized cases.
- Use `IClassFixture<T>` for expensive shared resources (containers, HTTP clients) — created once per test class.
- Use `NSubstitute` for mocking interfaces — call `Substitute.For<IMyInterface>()`.
- Use `Testcontainers` for real DB integration tests — never mock the DB for integration tests.
- Use `WebApplicationFactory<Program>` for API tests — replace real services with test doubles via `ConfigureServices`.
- Always use `FluentAssertions` for readable assertions — avoid raw `Assert.Equal`.
