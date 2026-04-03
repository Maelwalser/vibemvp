# C# + Carter Skill Guide

## Project Layout

```
ServiceName/
├── ServiceName.csproj
├── Program.cs               # app.MapCarter() registration
├── Modules/                 # ICarterModule implementations (one per feature)
│   ├── UserModule.cs
│   └── ItemModule.cs
├── Services/
└── Models/
```

## Dependencies

```xml
<PackageReference Include="Carter" Version="8.*" />
```

## Program.cs

```csharp
// Program.cs
var builder = WebApplication.CreateBuilder(args);

builder.Services.AddCarter();
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();
builder.Services.AddScoped<IUserService, UserService>();

var app = builder.Build();

app.UseSwagger();
app.UseSwaggerUI();
app.UseAuthentication();
app.UseAuthorization();

app.MapCarter();  // Discovers and registers all ICarterModule implementations

app.Run();
```

## ICarterModule (Module-per-Feature)

```csharp
// Modules/UserModule.cs
using Carter;
using Microsoft.AspNetCore.Http.HttpResults;

public class UserModule : ICarterModule
{
    public void AddRoutes(IEndpointRouteBuilder app)
    {
        var group = app.MapGroup("/api/users")
            .WithTags("Users")
            .RequireAuthorization();

        group.MapGet("/", GetAll);
        group.MapGet("/{id:int}", GetById);
        group.MapPost("/", Create).WithName("CreateUser");
        group.MapPut("/{id:int}", Update);
        group.MapDelete("/{id:int}", Delete);
    }

    private static async Task<Ok<IEnumerable<UserResponse>>> GetAll(
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

## Carter + Validation (FluentValidation)

```xml
<PackageReference Include="FluentValidation.AspNetCore" Version="11.*" />
```

```csharp
// Models/UserCreateRequest.cs
using FluentValidation;

public record UserCreateRequest(string Name, string Email);

public class UserCreateRequestValidator : AbstractValidator<UserCreateRequest>
{
    public UserCreateRequestValidator()
    {
        RuleFor(x => x.Name).NotEmpty().MinimumLength(1);
        RuleFor(x => x.Email).NotEmpty().EmailAddress();
    }
}

// Program.cs
builder.Services.AddValidatorsFromAssemblyContaining<UserCreateRequestValidator>();
```

```csharp
// Apply validation as endpoint filter in the module
group.MapPost("/", Create)
    .WithName("CreateUser")
    .AddEndpointFilter<ValidationFilter<UserCreateRequest>>();
```

## Carter with Response Negotiation

```csharp
// Carter supports IResponseNegotiator for content-type negotiation
// Default: JSON. Override for XML or custom media types.
public class XmlResponseNegotiator : IResponseNegotiator
{
    public bool CanHandle(MediaTypeHeaderValue accept)
        => accept.MediaType.ToString().Contains("xml", StringComparison.OrdinalIgnoreCase);

    public async Task Handle<T>(HttpRequest req, HttpResponse res, T model, CancellationToken ct)
    {
        res.ContentType = "application/xml";
        await res.WriteAsync(SerializeToXml(model), ct);
    }
}

// Register in Program.cs
builder.Services.AddSingleton<IResponseNegotiator, XmlResponseNegotiator>();
```

## Error Handling

- Use `TypedResults.Problem(...)` for RFC 7807 error responses inside module handlers.
- Register a global `IExceptionHandler` in `Program.cs` for unhandled exceptions.
- Carter delegates error handling to the standard ASP.NET Core middleware pipeline — add `app.UseExceptionHandler(...)` before `app.MapCarter()`.

## Key Rules

- One `ICarterModule` per feature/resource — never put unrelated routes in the same module.
- `app.MapCarter()` auto-discovers all `ICarterModule` implementations via DI scanning — no manual registration needed.
- Use `MapGroup()` inside `AddRoutes()` to apply shared middleware (auth, filters) to the module's routes.
- Always use `TypedResults` for OpenAPI schema accuracy.
- Carter modules are regular classes — inject services via static handler parameters, not constructor injection.
- Keep `Program.cs` minimal: only DI registration and middleware pipeline — all routes live in modules.
