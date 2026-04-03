# PHP + Slim 4 Skill Guide

## Project Layout

```
public/
└── index.php           # Entry point
src/
├── Action/             # Request handlers (thin controllers)
│   └── UserAction.php
├── Middleware/
│   └── AuthMiddleware.php
├── Repository/
│   └── UserRepository.php
└── Domain/
    └── User.php
config/
├── container.php       # PHP-DI definitions
└── settings.php
composer.json
```

## Bootstrap (public/index.php)

```php
<?php

use DI\ContainerBuilder;
use Slim\Factory\AppFactory;
use Slim\Middleware\ErrorMiddleware;

require __DIR__ . '/../vendor/autoload.php';

// Build DI container
$builder = new ContainerBuilder();
$builder->addDefinitions(__DIR__ . '/../config/container.php');
$container = $builder->build();

// Create Slim app from container
AppFactory::setContainer($container);
$app = AppFactory::create();

// Add middleware (order: last-added runs first)
$app->addRoutingMiddleware();
$app->addBodyParsingMiddleware();       // parses JSON / form / XML bodies

$errorMiddleware = $app->addErrorMiddleware(
    displayErrorDetails: (bool) getenv('APP_DEBUG'),
    logErrors:           true,
    logErrorDetails:     true,
);

// Register routes
(require __DIR__ . '/../config/routes.php')($app);

$app->run();
```

## Route Registration

```php
<?php
// config/routes.php

use Slim\App;
use App\Action\UserAction;
use App\Middleware\AuthMiddleware;

return function (App $app): void {
    $app->get('/health', function ($request, $response) {
        $response->getBody()->write(json_encode(['status' => 'ok']));
        return $response->withHeader('Content-Type', 'application/json');
    });

    $app->group('/api/v1', function ($group) {
        $group->get('/users',        [UserAction::class, 'list']);
        $group->get('/users/{id}',   [UserAction::class, 'show']);
        $group->post('/users',       [UserAction::class, 'create']);
        $group->patch('/users/{id}', [UserAction::class, 'update']);
        $group->delete('/users/{id}', [UserAction::class, 'delete']);
    })->add(AuthMiddleware::class);
};
```

## Action (Handler)

```php
<?php

namespace App\Action;

use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use App\Repository\UserRepository;

class UserAction
{
    public function __construct(
        private readonly UserRepository $repository,
    ) {}

    public function list(Request $request, Response $response): Response
    {
        $params = $request->getQueryParams();
        $page   = (int) ($params['page'] ?? 1);

        $users = $this->repository->paginate($page, 25);
        return $this->json($response, ['data' => $users]);
    }

    public function show(Request $request, Response $response, array $args): Response
    {
        $user = $this->repository->findById((int) $args['id']);
        if ($user === null) {
            return $this->json($response, ['error' => 'Not found'], 404);
        }
        return $this->json($response, ['data' => $user]);
    }

    public function create(Request $request, Response $response): Response
    {
        $body = (array) $request->getParsedBody();

        $errors = $this->validate($body, ['name', 'email']);
        if (!empty($errors)) {
            return $this->json($response, ['errors' => $errors], 422);
        }

        $user = $this->repository->create($body);
        return $this->json($response, ['data' => $user], 201);
    }

    public function update(Request $request, Response $response, array $args): Response
    {
        $user = $this->repository->findById((int) $args['id']);
        if ($user === null) {
            return $this->json($response, ['error' => 'Not found'], 404);
        }

        $body    = (array) $request->getParsedBody();
        $updated = $this->repository->update($user['id'], $body);
        return $this->json($response, ['data' => $updated]);
    }

    public function delete(Request $request, Response $response, array $args): Response
    {
        $user = $this->repository->findById((int) $args['id']);
        if ($user === null) {
            return $this->json($response, ['error' => 'Not found'], 404);
        }

        $this->repository->delete($user['id']);
        return $response->withStatus(204);
    }

    private function json(Response $response, mixed $data, int $status = 200): Response
    {
        $response->getBody()->write(json_encode($data, JSON_THROW_ON_ERROR));
        return $response
            ->withHeader('Content-Type', 'application/json')
            ->withStatus($status);
    }

    private function validate(array $data, array $required): array
    {
        $errors = [];
        foreach ($required as $field) {
            if (empty($data[$field])) {
                $errors[] = "$field is required";
            }
        }
        return $errors;
    }
}
```

## PSR-15 Middleware

```php
<?php

namespace App\Middleware;

use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface as RequestHandler;
use Slim\Psr7\Response as SlimResponse;

class AuthMiddleware implements MiddlewareInterface
{
    public function process(Request $request, RequestHandler $handler): Response
    {
        $token = $this->extractBearerToken($request);

        if ($token === null || !$this->isValid($token)) {
            $response = new SlimResponse();
            $response->getBody()->write(json_encode(['error' => 'Unauthorized']));
            return $response
                ->withHeader('Content-Type', 'application/json')
                ->withStatus(401);
        }

        // Pass enriched request to next middleware
        $request = $request->withAttribute('user_id', $this->resolveUserId($token));
        return $handler->handle($request);
    }

    private function extractBearerToken(Request $request): ?string
    {
        $header = $request->getHeaderLine('Authorization');
        if (str_starts_with($header, 'Bearer ')) {
            return substr($header, 7);
        }
        return null;
    }

    private function isValid(string $token): bool
    {
        return hash_equals($_ENV['API_TOKEN'] ?? '', $token);
    }

    private function resolveUserId(string $token): int
    {
        // Lookup logic here
        return 1;
    }
}
```

## PHP-DI Container Definitions

```php
<?php
// config/container.php

use DI\ContainerBuilder;
use App\Repository\UserRepository;
use PDO;

return [
    PDO::class => function () {
        $dsn = $_ENV['DATABASE_URL'] ?? throw new RuntimeException('DATABASE_URL not set');
        return new PDO($dsn, options: [
            PDO::ATTR_ERRMODE            => PDO::ERRMODE_EXCEPTION,
            PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
        ]);
    },

    UserRepository::class => DI\autowire(),
];
```

## ErrorMiddleware for Centralized Error Handling

```php
<?php
// Custom error handler registered on ErrorMiddleware

use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Slim\Exception\HttpNotFoundException;
use Slim\Middleware\ErrorMiddleware;

$errorMiddleware->setDefaultErrorHandler(
    function (Request $request, Throwable $exception, bool $displayErrorDetails) use ($app): Response {
        $response = $app->getResponseFactory()->createResponse();
        $statusCode = 500;

        if ($exception instanceof HttpNotFoundException) {
            $statusCode = 404;
        }

        $payload = ['error' => $exception->getMessage()];
        if ($displayErrorDetails) {
            $payload['trace'] = $exception->getTraceAsString();
        }

        $response->getBody()->write(json_encode($payload, JSON_THROW_ON_ERROR));
        return $response
            ->withHeader('Content-Type', 'application/json')
            ->withStatus($statusCode);
    }
);
```

## Environment Variables

```php
// Always read from environment — never hardcode
$dbUrl  = $_ENV['DATABASE_URL'] ?? throw new RuntimeException('DATABASE_URL not set');
$secret = $_ENV['APP_SECRET']   ?? throw new RuntimeException('APP_SECRET not set');
```

## Error Handling

- PSR-7 Response objects are immutable — every `with*` method returns a new instance.
- Use `$request->getParsedBody()` for JSON bodies (requires `addBodyParsingMiddleware()`).
- Throw `Slim\Exception\HttpNotFoundException` / `HttpUnauthorizedException` from actions — ErrorMiddleware catches them.
- Never output directly — always write to `$response->getBody()` and return the response.
