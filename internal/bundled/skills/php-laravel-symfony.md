# PHP + Laravel / Symfony Skill Guide

## Laravel Project Layout

```
app/
├── Http/
│   ├── Controllers/
│   │   └── Api/V1/UserController.php
│   ├── Middleware/
│   │   └── EnsureApiToken.php
│   └── Requests/
│       └── StoreUserRequest.php
├── Models/
│   └── User.php
└── Providers/
    └── AppServiceProvider.php
routes/
├── api.php
└── web.php
```

## Laravel Routes

```php
// routes/api.php
use Illuminate\Support\Facades\Route;
use App\Http\Controllers\Api\V1\UserController;

Route::prefix('v1')->middleware(['auth:sanctum'])->group(function () {
    Route::apiResource('users', UserController::class);

    Route::prefix('admin')->middleware(['role:admin'])->group(function () {
        Route::get('stats', [AdminController::class, 'stats']);
    });
});

Route::get('/health', fn () => response()->json(['status' => 'ok']));
```

## Eloquent Model

```php
<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\Factories\HasFactory;
use Illuminate\Database\Eloquent\SoftDeletes;

class User extends Model
{
    use HasFactory, SoftDeletes;

    protected $fillable = ['name', 'email', 'role', 'organization_id'];

    protected $hidden = ['password', 'remember_token'];

    protected $casts = [
        'email_verified_at' => 'datetime',
        'settings'          => 'array',
        'is_active'         => 'boolean',
    ];

    // Relationships
    public function organization(): BelongsTo
    {
        return $this->belongsTo(Organization::class);
    }

    public function posts(): HasMany
    {
        return $this->hasMany(Post::class);
    }

    // Scopes
    public function scopeActive($query)
    {
        return $query->where('is_active', true);
    }

    public function scopeRole($query, string $role)
    {
        return $query->where('role', $role);
    }
}
```

## Form Request Validation

```php
<?php

namespace App\Http\Requests;

use Illuminate\Foundation\Http\FormRequest;

class StoreUserRequest extends FormRequest
{
    public function authorize(): bool
    {
        return $this->user()->can('create', User::class);
    }

    public function rules(): array
    {
        return [
            'name'  => ['required', 'string', 'max:100'],
            'email' => ['required', 'email', 'unique:users,email'],
            'role'  => ['required', 'in:admin,member,viewer'],
        ];
    }

    public function messages(): array
    {
        return [
            'email.unique' => 'This email address is already registered.',
        ];
    }
}
```

## Laravel Controller

```php
<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Controller;
use App\Http\Requests\StoreUserRequest;
use App\Http\Requests\UpdateUserRequest;
use App\Models\User;

class UserController extends Controller
{
    public function index()
    {
        $users = User::active()->paginate(25);
        return response()->json(['data' => $users]);
    }

    public function show(User $user)
    {
        return response()->json(['data' => $user]);
    }

    public function store(StoreUserRequest $request)
    {
        $user = User::create($request->validated());
        return response()->json(['data' => $user], 201);
    }

    public function update(UpdateUserRequest $request, User $user)
    {
        $user->update($request->validated());
        return response()->json(['data' => $user->fresh()]);
    }

    public function destroy(User $user)
    {
        $user->delete();
        return response()->noContent();
    }
}
```

## Laravel Middleware

```php
<?php

namespace App\Http\Middleware;

use Closure;
use Illuminate\Http\Request;

class EnsureApiToken
{
    public function handle(Request $request, Closure $next): mixed
    {
        $token = $request->bearerToken();
        if (!$token || !hash_equals(config('app.api_token'), $token)) {
            return response()->json(['error' => 'Unauthorized'], 401);
        }
        return $next($request);
    }
}

// Register in app/Http/Kernel.php or bootstrap/app.php (Laravel 11+):
// ->withMiddleware(function (Middleware $middleware) {
//     $middleware->alias(['api.token' => EnsureApiToken::class]);
// })
```

## Artisan Command

```php
<?php

namespace App\Console\Commands;

use Illuminate\Console\Command;

class SyncUsers extends Command
{
    protected $signature   = 'users:sync {--dry-run : Print changes without applying}';
    protected $description = 'Sync users from external source';

    public function handle(): int
    {
        $dryRun = $this->option('dry-run');
        $this->info('Starting sync...');

        // logic here

        $this->table(['ID', 'Email'], $rows);
        return self::SUCCESS;
    }
}
```

## Symfony Route and Controller

```php
<?php

namespace App\Controller;

use Symfony\Bundle\FrameworkBundle\Controller\AbstractController;
use Symfony\Component\HttpFoundation\JsonResponse;
use Symfony\Component\HttpFoundation\Request;
use Symfony\Component\Routing\Attribute\Route;

#[Route('/api/v1')]
class UserController extends AbstractController
{
    public function __construct(
        private readonly UserRepository $userRepository,
        private readonly EntityManagerInterface $em,
    ) {}

    #[Route('/users', methods: ['GET'])]
    public function index(): JsonResponse
    {
        $users = $this->userRepository->findAll();
        return $this->json(['data' => $users]);
    }

    #[Route('/users/{id}', methods: ['GET'])]
    public function show(int $id): JsonResponse
    {
        $user = $this->userRepository->find($id);
        if (!$user) {
            return $this->json(['error' => 'Not found'], 404);
        }
        return $this->json(['data' => $user]);
    }

    #[Route('/users', methods: ['POST'])]
    public function create(Request $request): JsonResponse
    {
        $data = json_decode($request->getContent(), true, 512, JSON_THROW_ON_ERROR);

        $user = new User();
        $user->setName($data['name']);
        $user->setEmail($data['email']);

        $this->em->persist($user);
        $this->em->flush();

        return $this->json(['data' => $user], 201);
    }
}
```

## Symfony Entity and Repository

```php
<?php

namespace App\Entity;

use App\Repository\UserRepository;
use Doctrine\ORM\Mapping as ORM;

#[ORM\Entity(repositoryClass: UserRepository::class)]
class User
{
    #[ORM\Id]
    #[ORM\GeneratedValue]
    #[ORM\Column]
    private ?int $id = null;

    #[ORM\Column(length: 100)]
    private string $name;

    #[ORM\Column(length: 180, unique: true)]
    private string $email;

    public function getId(): ?int   { return $this->id; }
    public function getName(): string { return $this->name; }
    public function setName(string $name): static { $this->name = $name; return $this; }
    public function getEmail(): string { return $this->email; }
    public function setEmail(string $email): static { $this->email = $email; return $this; }
}
```

## Symfony Security Voter

```php
<?php

namespace App\Security\Voter;

use Symfony\Component\Security\Core\Authorization\Voter\Voter;

class UserVoter extends Voter
{
    protected function supports(string $attribute, mixed $subject): bool
    {
        return in_array($attribute, ['VIEW', 'EDIT', 'DELETE']) && $subject instanceof User;
    }

    protected function voteOnAttribute(string $attribute, mixed $subject, TokenInterface $token): bool
    {
        $currentUser = $token->getUser();
        if (!$currentUser instanceof User) {
            return false;
        }

        return match ($attribute) {
            'VIEW'   => true,
            'EDIT'   => $currentUser === $subject || $currentUser->isAdmin(),
            'DELETE' => $currentUser->isAdmin(),
            default  => false,
        };
    }
}
```

## Error Handling

- Laravel: use `abort(404)` / `abort_if()` helpers in controllers; register custom exception renderers in `bootstrap/app.php`.
- Symfony: throw `NotFoundHttpException` / `AccessDeniedException`; register `kernel.exception` event listener for custom responses.
- Always return JSON error envelopes: `{ "error": "..." }` or `{ "errors": [...] }`.
- Never expose stack traces or internal messages to API consumers.
