# PHP + Laminas (Zend) Skill Guide

## Project Layout

```
module/
└── User/
    ├── config/
    │   └── module.config.php
    ├── src/
    │   ├── Controller/
    │   │   └── UserController.php
    │   ├── Factory/
    │   │   ├── UserControllerFactory.php
    │   │   └── UserServiceFactory.php
    │   ├── Form/
    │   │   └── UserForm.php
    │   ├── InputFilter/
    │   │   └── UserInputFilter.php
    │   ├── Model/
    │   │   └── User.php
    │   ├── Service/
    │   │   └── UserService.php
    │   └── Module.php
    └── view/
        └── user/
            └── user/
                └── index.phtml
```

## Module Config

```php
<?php
// module/User/config/module.config.php

use Laminas\Router\Http\Literal;
use Laminas\Router\Http\Segment;
use Laminas\ServiceManager\Factory\InvokableFactory;
use User\Controller\UserController;
use User\Factory\UserControllerFactory;
use User\Service\UserService;
use User\Factory\UserServiceFactory;

return [
    'router' => [
        'routes' => [
            'users' => [
                'type'    => Literal::class,
                'options' => [
                    'route'    => '/api/v1/users',
                    'defaults' => [
                        'controller' => UserController::class,
                        'action'     => 'index',
                    ],
                ],
                'may_terminate' => true,
                'child_routes'  => [
                    'id' => [
                        'type'    => Segment::class,
                        'options' => [
                            'route'       => '/[:id]',
                            'constraints' => ['id' => '[0-9]+'],
                            'defaults'    => ['action' => 'show'],
                        ],
                    ],
                ],
            ],
        ],
    ],

    'controllers' => [
        'factories' => [
            UserController::class => UserControllerFactory::class,
        ],
    ],

    'service_manager' => [
        'factories' => [
            UserService::class => UserServiceFactory::class,
        ],
        'invokables' => [
            // Services with no dependencies
        ],
    ],

    'view_manager' => [
        'strategies'           => ['ViewJsonStrategy'],   // enable JSON renderer
        'template_path_stack'  => [__DIR__ . '/../view'],
    ],
];
```

## Module Class

```php
<?php
// module/User/src/Module.php

namespace User;

use Laminas\ModuleManager\Feature\ConfigProviderInterface;

class Module implements ConfigProviderInterface
{
    public function getConfig(): array
    {
        return include __DIR__ . '/../config/module.config.php';
    }
}
```

## AbstractActionController

```php
<?php

namespace User\Controller;

use Laminas\Mvc\Controller\AbstractActionController;
use Laminas\View\Model\JsonModel;
use User\Service\UserService;

class UserController extends AbstractActionController
{
    public function __construct(
        private readonly UserService $userService,
    ) {}

    public function indexAction(): JsonModel
    {
        $page    = (int) ($this->params()->fromQuery('page', 1));
        $perPage = (int) ($this->params()->fromQuery('per_page', 25));

        $users = $this->userService->paginate($page, $perPage);
        return new JsonModel(['data' => $users]);
    }

    public function showAction(): JsonModel
    {
        $id   = (int) $this->params()->fromRoute('id');
        $user = $this->userService->findById($id);

        if ($user === null) {
            $this->response->setStatusCode(404);
            return new JsonModel(['error' => 'Not found']);
        }

        return new JsonModel(['data' => $user->toArray()]);
    }

    public function createAction(): JsonModel
    {
        $data   = json_decode($this->request->getContent(), true, 512, JSON_THROW_ON_ERROR);
        $filter = new \User\InputFilter\UserInputFilter();
        $filter->setData($data);

        if (!$filter->isValid()) {
            $this->response->setStatusCode(422);
            return new JsonModel(['errors' => $filter->getMessages()]);
        }

        $user = $this->userService->create($filter->getValues());
        $this->response->setStatusCode(201);
        return new JsonModel(['data' => $user->toArray()]);
    }

    public function updateAction(): JsonModel
    {
        $id   = (int) $this->params()->fromRoute('id');
        $user = $this->userService->findById($id);

        if ($user === null) {
            $this->response->setStatusCode(404);
            return new JsonModel(['error' => 'Not found']);
        }

        $data    = json_decode($this->request->getContent(), true, 512, JSON_THROW_ON_ERROR);
        $updated = $this->userService->update($id, $data);
        return new JsonModel(['data' => $updated->toArray()]);
    }

    public function deleteAction(): JsonModel
    {
        $id = (int) $this->params()->fromRoute('id');
        $this->userService->delete($id);
        $this->response->setStatusCode(204);
        return new JsonModel([]);
    }
}
```

## ServiceManager Factories

```php
<?php
// module/User/src/Factory/UserControllerFactory.php

namespace User\Factory;

use Interop\Container\ContainerInterface;
use Laminas\ServiceManager\Factory\FactoryInterface;
use User\Controller\UserController;
use User\Service\UserService;

class UserControllerFactory implements FactoryInterface
{
    public function __invoke(ContainerInterface $container, $requestedName, ?array $options = null): UserController
    {
        return new UserController(
            $container->get(UserService::class),
        );
    }
}

// InvokableFactory — use in module.config.php for zero-dependency services:
// 'invokables' => [SimpleService::class => SimpleService::class],
```

## InputFilter for Validation

```php
<?php

namespace User\InputFilter;

use Laminas\InputFilter\InputFilter;
use Laminas\Validator\EmailAddress;
use Laminas\Validator\InArray;
use Laminas\Validator\StringLength;
use Laminas\Filter\StringTrim;
use Laminas\Filter\StripTags;

class UserInputFilter extends InputFilter
{
    public function init(): void
    {
        $this->add([
            'name'       => 'name',
            'required'   => true,
            'filters'    => [
                ['name' => StringTrim::class],
                ['name' => StripTags::class],
            ],
            'validators' => [
                [
                    'name'    => StringLength::class,
                    'options' => ['min' => 1, 'max' => 100],
                ],
            ],
        ]);

        $this->add([
            'name'       => 'email',
            'required'   => true,
            'filters'    => [['name' => StringTrim::class]],
            'validators' => [
                ['name' => EmailAddress::class],
            ],
        ]);

        $this->add([
            'name'       => 'role',
            'required'   => true,
            'validators' => [
                [
                    'name'    => InArray::class,
                    'options' => ['haystack' => ['admin', 'member', 'viewer']],
                ],
            ],
        ]);
    }
}
```

## ViewModel (HTML Views)

```php
<?php
// In controller action returning HTML
use Laminas\View\Model\ViewModel;

public function indexAction(): ViewModel
{
    $users = $this->userService->findAll();

    $view = new ViewModel();
    $view->setVariable('users', $users);
    $view->setVariable('title', 'User List');
    $view->setTemplate('user/user/index');
    return $view;
}

// view/user/user/index.phtml
// <?= $this->escapeHtml($title) ?>
// <?php foreach ($users as $user): ?>
//   <p><?= $this->escapeHtml($user->getName()) ?></p>
// <?php endforeach; ?>
```

## Environment Variables

```php
// Read from environment or config — never hardcode credentials.
$dbHost = getenv('DB_HOST') ?: throw new \RuntimeException('DB_HOST not set');
$secret = getenv('APP_SECRET') ?: throw new \RuntimeException('APP_SECRET not set');
```

## Error Handling

- Use `$this->response->setStatusCode(N)` before returning `JsonModel` — never throw unhandled exceptions.
- Validate all input through `InputFilter` before passing to the service layer.
- Register a custom `ExceptionStrategy` in module config to convert uncaught exceptions to JSON error responses.
- Never expose internal exception messages or stack traces to the client.
