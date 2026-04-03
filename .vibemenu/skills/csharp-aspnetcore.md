# C# + ASP.NET Core Skill Guide

## Project Layout

```
ServiceName/
├── ServiceName.csproj
├── Program.cs               # Entry point, DI registration, middleware pipeline
├── appsettings.json
├── appsettings.Development.json
├── Controllers/             # [ApiController] classes
├── Services/                # Business logic interfaces + implementations
├── Repositories/            # Data access
├── Models/                  # Request/response DTOs
└── Middleware/              # Custom IMiddleware implementations
```

## .csproj Boilerplate

```xml
<Project Sdk="Microsoft.NET.Sdk.Web">
  <PropertyGroup>
    <TargetFramework>net9.0</TargetFramework>
    <Nullable>enable</Nullable>
    <ImplicitUsings>enable</ImplicitUsings>
  </PropertyGroup>
  <ItemGroup>
    <PackageReference Include="Microsoft.EntityFrameworkCore.Design" Version="9.*" />
    <PackageReference Include="Npgsql.EntityFrameworkCore.PostgreSQL" Version="9.*" />
  </ItemGroup>
</Project>
```

## Program.cs Setup

```csharp
// Program.cs
var builder = WebApplication.CreateBuilder(args);

builder.Services.AddControllers();
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();
builder.Services.AddScoped<IUserService, UserService>();
builder.Services.AddScoped<IUserRepository, UserRepository>();

var app = builder.Build();

if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}

app.UseHttpsRedirection();
app.UseAuthentication();
app.UseAuthorization();
app.UseMiddleware<RequestLoggingMiddleware>();
app.MapControllers();

app.Run();
```

## Controller Pattern

```csharp
// Controllers/UsersController.cs
using Microsoft.AspNetCore.Mvc;

[ApiController]
[Route("api/[controller]")]
public class UsersController : ControllerBase
{
    private readonly IUserService _svc;

    public UsersController(IUserService svc) => _svc = svc;

    [HttpGet]
    [ProducesResponseType(typeof(IEnumerable<UserResponse>), StatusCodes.Status200OK)]
    public async Task<IActionResult> GetAll(CancellationToken ct)
    {
        var users = await _svc.GetAllAsync(ct);
        return Ok(users);
    }

    [HttpGet("{id:int}")]
    [ProducesResponseType(typeof(UserResponse), StatusCodes.Status200OK)]
    [ProducesResponseType(StatusCodes.Status404NotFound)]
    public async Task<IActionResult> GetById(int id, CancellationToken ct)
    {
        var user = await _svc.GetByIdAsync(id, ct);
        return user is null ? NotFound() : Ok(user);
    }

    [HttpPost]
    [ProducesResponseType(typeof(UserResponse), StatusCodes.Status201Created)]
    [ProducesResponseType(typeof(ValidationProblemDetails), StatusCodes.Status400BadRequest)]
    public async Task<IActionResult> Create([FromBody] UserCreateRequest req, CancellationToken ct)
    {
        var user = await _svc.CreateAsync(req, ct);
        return CreatedAtAction(nameof(GetById), new { id = user.Id }, user);
    }

    [HttpPut("{id:int}")]
    [ProducesResponseType(StatusCodes.Status204NoContent)]
    [ProducesResponseType(StatusCodes.Status404NotFound)]
    public async Task<IActionResult> Update(int id, [FromBody] UserUpdateRequest req, CancellationToken ct)
    {
        var updated = await _svc.UpdateAsync(id, req, ct);
        return updated ? NoContent() : NotFound();
    }

    [HttpDelete("{id:int}")]
    [ProducesResponseType(StatusCodes.Status204NoContent)]
    public async Task<IActionResult> Delete(int id, CancellationToken ct)
    {
        await _svc.DeleteAsync(id, ct);
        return NoContent();
    }
}
```

## Model Binding

```csharp
// Models/UserCreateRequest.cs
using System.ComponentModel.DataAnnotations;

public record UserCreateRequest(
    [Required, MinLength(1)] string Name,
    [Required, EmailAddress] string Email,
    [Range(0, 150)] int Age
);

public record UserResponse(int Id, string Name, string Email, DateTime CreatedAt);
```

## Custom Middleware

```csharp
// Middleware/RequestLoggingMiddleware.cs
public class RequestLoggingMiddleware : IMiddleware
{
    private readonly ILogger<RequestLoggingMiddleware> _logger;

    public RequestLoggingMiddleware(ILogger<RequestLoggingMiddleware> logger)
        => _logger = logger;

    public async Task InvokeAsync(HttpContext context, RequestDelegate next)
    {
        _logger.LogInformation("{Method} {Path}", context.Request.Method, context.Request.Path);
        await next(context);
        _logger.LogInformation("Response {StatusCode}", context.Response.StatusCode);
    }
}
```

## Configuration

```csharp
// appsettings.json
{
  "ConnectionStrings": {
    "Postgres": "Host=localhost;Database=mydb;Username=app;Password=secret"
  },
  "App": {
    "PageSize": 20
  }
}

// Typed settings
public record AppSettings(int PageSize);

// Registration in Program.cs
builder.Services.Configure<AppSettings>(builder.Configuration.GetSection("App"));

// Injection
public class MyService(IOptions<AppSettings> opts)
{
    private readonly AppSettings _settings = opts.Value;
}
```

## Error Handling

- Return `IActionResult` with typed responses (`Ok()`, `NotFound()`, `BadRequest()`).
- `[ApiController]` automatically returns `ValidationProblemDetails` on model validation failure.
- Use `ProblemDetails` (RFC 7807) for error responses — call `Problem(title, detail, statusCode)`.
- Register a global exception handler with `app.UseExceptionHandler("/error")` or `IProblemDetailsService`.

## Key Rules

- Every controller must have `[ApiController]` and `[Route]` attributes.
- Use `[ProducesResponseType]` on every action for accurate OpenAPI docs.
- Inject dependencies via constructor — never use `ServiceLocator` or `HttpContext.RequestServices`.
- Use `CancellationToken` on every async action and pass it down to DB/HTTP calls.
- Use `record` types for immutable request/response DTOs.
- Prefer `IOptions<T>` over raw `IConfiguration` for strongly-typed config access.
