# PHP + PHPUnit Skill Guide

## Project Layout

```
tests/
├── Unit/
│   ├── Service/
│   │   └── UserServiceTest.php
│   └── Model/
│       └── UserTest.php
├── Integration/
│   └── Repository/
│       └── UserRepositoryTest.php
├── Feature/
│   └── UserApiTest.php
└── bootstrap.php
phpunit.xml
```

## phpunit.xml

```xml
<?xml version="1.0" encoding="UTF-8"?>
<phpunit xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:noNamespaceSchemaLocation="vendor/phpunit/phpunit/phpunit.xsd"
         bootstrap="tests/bootstrap.php"
         colors="true">
    <testsuites>
        <testsuite name="Unit">
            <directory>tests/Unit</directory>
        </testsuite>
        <testsuite name="Integration">
            <directory>tests/Integration</directory>
        </testsuite>
        <testsuite name="Feature">
            <directory>tests/Feature</directory>
        </testsuite>
    </testsuites>
    <coverage>
        <include>
            <directory>src</directory>
        </include>
    </coverage>
</phpunit>
```

## Basic TestCase

```php
<?php

namespace Tests\Unit\Service;

use PHPUnit\Framework\TestCase;
use App\Service\UserService;
use App\Repository\UserRepository;

class UserServiceTest extends TestCase
{
    private UserRepository $repository;
    private UserService    $service;

    protected function setUp(): void
    {
        parent::setUp();

        $this->repository = $this->createMock(UserRepository::class);
        $this->service    = new UserService($this->repository);
    }

    protected function tearDown(): void
    {
        // Clean up resources if needed
        parent::tearDown();
    }

    public function test_finds_user_by_id(): void
    {
        $expected = ['id' => 1, 'name' => 'Alice', 'email' => 'alice@example.com'];

        $this->repository
            ->expects($this->once())
            ->method('findById')
            ->with(1)
            ->willReturn($expected);

        $result = $this->service->findById(1);

        $this->assertSame($expected, $result);
    }

    public function test_returns_null_when_user_not_found(): void
    {
        $this->repository->method('findById')->willReturn(null);

        $result = $this->service->findById(99);

        $this->assertNull($result);
    }
}
```

## Data Providers

```php
<?php

namespace Tests\Unit;

use PHPUnit\Framework\TestCase;
use PHPUnit\Framework\Attributes\DataProvider;
use App\Validator\EmailValidator;

class EmailValidatorTest extends TestCase
{
    private EmailValidator $validator;

    protected function setUp(): void
    {
        $this->validator = new EmailValidator();
    }

    #[DataProvider('validEmailProvider')]
    public function test_accepts_valid_emails(string $email): void
    {
        $this->assertTrue($this->validator->isValid($email));
    }

    #[DataProvider('invalidEmailProvider')]
    public function test_rejects_invalid_emails(string $email): void
    {
        $this->assertFalse($this->validator->isValid($email));
    }

    public static function validEmailProvider(): array
    {
        return [
            'simple address'      => ['user@example.com'],
            'with plus sign'      => ['user+tag@example.com'],
            'subdomain'           => ['user@mail.example.com'],
        ];
    }

    public static function invalidEmailProvider(): array
    {
        return [
            'missing @'           => ['notanemail'],
            'missing domain'      => ['user@'],
            'missing local part'  => ['@example.com'],
            'empty string'        => [''],
        ];
    }
}
```

## Mocking with createMock / getMockBuilder

```php
<?php

namespace Tests\Unit\Service;

use PHPUnit\Framework\TestCase;
use App\Service\PaymentService;
use App\Gateway\StripeGateway;

class PaymentServiceTest extends TestCase
{
    public function test_charges_successfully(): void
    {
        // Simple mock
        $gateway = $this->createMock(StripeGateway::class);
        $gateway->method('charge')
                ->willReturn(['id' => 'ch_123', 'status' => 'succeeded']);

        $service = new PaymentService($gateway);
        $result  = $service->charge(500, 'usd');

        $this->assertTrue($result->isSuccessful());
    }

    public function test_raises_exception_when_gateway_fails(): void
    {
        $gateway = $this->getMockBuilder(StripeGateway::class)
                        ->disableOriginalConstructor()
                        ->onlyMethods(['charge'])
                        ->getMock();

        $gateway->method('charge')
                ->willThrowException(new \RuntimeException('Card declined'));

        $service = new PaymentService($gateway);

        $this->expectException(\App\Exception\PaymentException::class);
        $this->expectExceptionMessage('Card declined');

        $service->charge(500, 'usd');
    }

    public function test_calls_gateway_with_correct_params(): void
    {
        $gateway = $this->createMock(StripeGateway::class);
        $gateway->expects($this->once())
                ->method('charge')
                ->with(
                    $this->equalTo(500),
                    $this->equalTo('usd'),
                )
                ->willReturn(['id' => 'ch_456', 'status' => 'succeeded']);

        $service = new PaymentService($gateway);
        $service->charge(500, 'usd');
    }
}
```

## Assertions Reference

```php
// Equality
$this->assertSame($expected, $actual);           // strict (===)
$this->assertEquals($expected, $actual);         // loose (==)
$this->assertNotSame($expected, $actual);

// Type
$this->assertInstanceOf(User::class, $result);
$this->assertIsArray($result);
$this->assertIsString($result);
$this->assertIsInt($result);
$this->assertNull($result);
$this->assertNotNull($result);

// Boolean
$this->assertTrue($condition);
$this->assertFalse($condition);

// Collections
$this->assertCount(3, $collection);
$this->assertEmpty($collection);
$this->assertNotEmpty($collection);
$this->assertContains('needle', $haystack);
$this->assertArrayHasKey('key', $array);
$this->assertArrayNotHasKey('key', $array);

// Strings
$this->assertStringContainsString('needle', $haystack);
$this->assertStringStartsWith('prefix', $string);
$this->assertMatchesRegularExpression('/pattern/', $string);

// Exceptions
$this->expectException(\InvalidArgumentException::class);
$this->expectExceptionMessage('Expected message');
$this->expectExceptionCode(42);
```

## Integration Test with Database

```php
<?php

namespace Tests\Integration\Repository;

use PHPUnit\Framework\TestCase;
use App\Repository\UserRepository;
use PDO;

class UserRepositoryTest extends TestCase
{
    private PDO            $pdo;
    private UserRepository $repository;

    protected function setUp(): void
    {
        $this->pdo = new PDO(
            $_ENV['TEST_DATABASE_URL'] ?? throw new \RuntimeException('TEST_DATABASE_URL not set'),
            options: [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]
        );
        $this->pdo->beginTransaction();  // wrap each test in a transaction

        $this->repository = new UserRepository($this->pdo);
    }

    protected function tearDown(): void
    {
        $this->pdo->rollBack();           // rollback after each test
    }

    public function test_creates_and_retrieves_user(): void
    {
        $created = $this->repository->create([
            'name'  => 'Bob',
            'email' => 'bob@example.com',
            'role'  => 'member',
        ]);

        $found = $this->repository->findById($created['id']);

        $this->assertSame('Bob', $found['name']);
        $this->assertSame('bob@example.com', $found['email']);
    }
}
```

## Code Coverage Annotation

```php
<?php

namespace Tests\Unit;

use PHPUnit\Framework\TestCase;
use PHPUnit\Framework\Attributes\CoversClass;
use PHPUnit\Framework\Attributes\CoversMethod;
use App\Service\UserService;

#[CoversClass(UserService::class)]
class UserServiceCoverageTest extends TestCase
{
    // Tests here count toward UserService coverage
}
```

## Error Handling in Tests

- Use `$this->expectException()` before the action under test — never wrap in `try/catch`.
- Prefer `assertSame` over `assertEquals` for type safety.
- Avoid `@dataProvider` annotation syntax in favor of `#[DataProvider]` attribute (PHPUnit 10+).
- Use `setUp` for shared state — never carry state between test methods.
- Wrap integration tests in transactions and roll back in `tearDown` to keep the database clean.
