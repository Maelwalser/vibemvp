# C# + .NET Minimal APIs Skill Guide

## Project Layout

```
ServiceName/
├── ServiceName.csproj
├── Program.cs               # All route registrations + DI
├── appsettings.json
├── Endpoints/               # Extension methods grouping related routes
│   ├── UserEndpoints.cs
│   └── ItemEndpoints.cs
├── Services/
├── Models/                  # Request/response records
└── Filters/                 # IEndpointFilter implementations
```

## App Setup

```csharp
// Program.cs
var builder = WebApplication.CreateBuilder(args);

builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();
builder.Services.AddScoped<IUserService, UserService>();

var app = builder.Build();

app.UseSwagger();
app.UseSwaggerUI();

// Map endpoint groups from extension methods
app.MapUserEndpoints();
app.MapItemEndpoints();

app.MapGet("/health", () => Results.Ok(new { status = "ok" }))
   .WithTags("health")
   .AllowAnonymous();

app.Run();
```

## Endpoint Groups (MapGroup)

```csharp
// Endpoints/UserEndpoints.cs
public static class UserEndpoints
{
    public static void MapUserEndpoints(this WebApplication app)
    {
        var group = app.MapGroup("/api/users")
            .WithTags("Users")
            .RequireAuthorization()
            .AddEndpointFilter<ValidationFilter<UserCreateRequest>>();

        group.MapGet("/", GetAll);
        group.MapGet("/{id:int}", GetById);
        group.MapPost("/", Create).WithName("CreateUser");
        group.MapPut("/{id:int}", Update);
        group.MapDelete("/{id:int}", Delete);
    }

    private static async Task<Results<Ok<IEnumerable<UserResponse>>, NotFound>> GetAll(
        IUserService svc,
        CancellationToken ct)
    {
        var users = await svc.GetAllAsync(ct);
        return TypedResults.Ok(users);
    }

    private static async Task<Results<Ok<UserResponse>, NotFound>> GetById(
        int id,
        IUserService svc,
        CancellationToken ct)
    {
        var user = await svc.GetByIdAsync(id, ct);
        return user is not null
            ? TypedResults.Ok(user)
            : TypedResults.NotFound();
    }

    private static async Task<Results<CreatedAtRoute<UserResponse>, BadRequest<string>>> Create(
        UserCreateRequest req,
        IUserService svc,
        CancellationToken ct)
    {
        var user = await svc.CreateAsync(req, ct);
        return TypedResults.CreatedAtRoute(user, "CreateUser", new { id = user.Id });
    }

    private static async Task<Results<NoContent, NotFound>> Update(
        int id,
        UserUpdateRequest req,
        IUserService svc,
        CancellationToken ct)
    {
        var updated = await svc.UpdateAsync(id, req, ct);
        return updated ? TypedResults.NoContent() : TypedResults.NotFound();
    }

    private static async Task<NoContent> Delete(int id, IUserService svc, CancellationToken ct)
    {
        await svc.DeleteAsync(id, ct);
        return TypedResults.NoContent();
    }
}
```

## TypedResults

```csharp
// Always prefer TypedResults over Results for OpenAPI schema generation
Results.Ok(data)          // untyped — avoids
TypedResults.Ok(data)     // typed — generates schema

// Common typed results
TypedResults.Ok(payload)
TypedResults.Created(uri, payload)
TypedResults.CreatedAtRoute(payload, routeName, routeValues)
TypedResults.NoContent()
TypedResults.NotFound()
TypedResults.BadRequest(errorMessage)
TypedResults.UnprocessableEntity(validationErrors)
TypedResults.Unauthorized()
TypedResults.Problem(detail, title: "Error", statusCode: 500)
```

## Endpoint Filters

```csharp
// Filters/ValidationFilter.cs
using System.ComponentModel.DataAnnotations;

public class ValidationFilter<T> : IEndpointFilter where T : class
{
    public async ValueTask<object?> InvokeAsync(
        EndpointFilterInvocationContext context,
        EndpointFilterDelegate next)
    {
        var argument = context.Arguments.OfType<T>().FirstOrDefault();
        if (argument is null)
            return Results.BadRequest("Missing request body");

        var errors = new List<ValidationResult>();
        if (!Validator.TryValidateObject(argument, new ValidationContext(argument), errors, true))
        {
            return Results.ValidationProblem(
                errors.ToDictionary(e => e.MemberNames.First(), e => new[] { e.ErrorMessage! })
            );
        }

        return await next(context);
    }
}
```

## Parameter Binding

```csharp
// Route parameter
app.MapGet("/users/{id:int}", (int id) => ...);

// Query string
app.MapGet("/users", (string? search, int page = 1) => ...);

// Request body (automatic from content-type)
app.MapPost("/users", (UserCreateRequest req) => ...);

// From services (DI)
app.MapGet("/users", (IUserService svc) => ...);

// From headers
app.MapGet("/me", ([FromHeader(Name = "X-User-Id")] string userId) => ...);

// HttpContext
app.MapGet("/info", (HttpContext ctx) => ctx.Connection.RemoteIpAddress?.ToString());
```

## Models

```csharp
using System.ComponentModel.DataAnnotations;

public record UserCreateRequest(
    [Required, MinLength(1)] string Name,
    [Required, EmailAddress] string Email
);

public record UserUpdateRequest([MinLength(1)] string? Name);
public record UserResponse(int Id, string Name, string Email, DateTime CreatedAt);
```

## Error Handling

- Use `TypedResults.Problem(...)` for structured RFC 7807 error responses.
- Add `app.UseExceptionHandler(...)` or a global `IExceptionHandler` for unhandled exceptions.
- Return union types `Results<T1, T2>` to document all possible responses in OpenAPI.

## Key Rules

- Use `TypedResults` (not `Results`) for all returns — this generates correct OpenAPI schemas.
- Group related endpoints with `app.MapGroup()` — don't scatter them across `Program.cs`.
- Use extension methods (`MapUserEndpoints()`) to keep `Program.cs` concise.
- Apply `IEndpointFilter` for cross-cutting concerns (validation, auth, logging) on groups, not inline.
- Always pass `CancellationToken` through to service and DB calls.
- Use `record` types for immutable DTOs with built-in equality.
