# Testing: Unit Tests Skill Guide

## Overview

Unit test patterns for Jest/Vitest (TypeScript), pytest (Python), Go testing package, JUnit 5 (Java), and xUnit (.NET).

## Jest / Vitest (TypeScript)

```typescript
// vitest.config.ts
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'node',
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html', 'lcov'],
      thresholds: { lines: 80, functions: 80, branches: 80 },
    },
    globals: true,
  },
});
```

```typescript
// user.service.test.ts
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { UserService } from './user.service';
import { UserRepository } from './user.repository';

vi.mock('./user.repository');

describe('UserService', () => {
  let service: UserService;
  let repo: UserRepository;

  beforeEach(() => {
    repo = new UserRepository() as vi.Mocked<UserRepository>;
    service = new UserService(repo);
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  describe('createUser', () => {
    it('creates a user and returns it', async () => {
      const mockUser = { id: '1', email: 'a@b.com', name: 'Alice' };
      vi.spyOn(repo, 'insert').mockResolvedValue(mockUser);

      const result = await service.createUser({ email: 'a@b.com', name: 'Alice' });

      expect(result).toEqual(mockUser);
      expect(repo.insert).toHaveBeenCalledWith({ email: 'a@b.com', name: 'Alice' });
      expect(repo.insert).toHaveBeenCalledTimes(1);
    });

    it('throws if email already exists', async () => {
      vi.spyOn(repo, 'insert').mockRejectedValue(new Error('duplicate key'));

      await expect(service.createUser({ email: 'a@b.com', name: 'Alice' }))
        .rejects.toThrow('duplicate key');
    });
  });
});
```

### Mock Return Values

```typescript
const mockFn = vi.fn();
mockFn.mockReturnValue('sync value');
mockFn.mockResolvedValue('async value');
mockFn.mockRejectedValue(new Error('oops'));
mockFn.mockReturnValueOnce('first call only');

// Module mock
vi.mock('../email', () => ({
  sendEmail: vi.fn().mockResolvedValue({ id: 'msg_123' }),
}));
```

## pytest (Python)

```python
# conftest.py
import pytest
from unittest.mock import AsyncMock, MagicMock

@pytest.fixture
def mock_repo():
    repo = MagicMock()
    repo.insert = AsyncMock(return_value={"id": "1", "email": "a@b.com"})
    return repo

@pytest.fixture
def user_service(mock_repo):
    from app.services.user import UserService
    return UserService(repo=mock_repo)
```

```python
# test_user_service.py
import pytest
from unittest.mock import AsyncMock, patch

class TestUserService:
    async def test_create_user_returns_user(self, user_service, mock_repo):
        result = await user_service.create_user(email="a@b.com", name="Alice")

        assert result["email"] == "a@b.com"
        mock_repo.insert.assert_called_once_with(email="a@b.com", name="Alice")

    @pytest.mark.parametrize("email,name,expected_error", [
        ("", "Alice", "email is required"),
        ("not-an-email", "Alice", "invalid email"),
        ("a@b.com", "", "name is required"),
    ])
    async def test_create_user_validation(self, user_service, email, name, expected_error):
        with pytest.raises(ValueError, match=expected_error):
            await user_service.create_user(email=email, name=name)

    def test_parse_config_uses_temp_path(self, tmp_path):
        config_file = tmp_path / "config.json"
        config_file.write_text('{"env": "test"}')

        from app.config import parse_config
        config = parse_config(config_file)
        assert config["env"] == "test"

    def test_patch_external(self, monkeypatch):
        monkeypatch.setattr("app.email.send", lambda *args: {"id": "mocked"})
        # ... test code that calls send internally ...
```

```ini
# pytest.ini
[pytest]
asyncio_mode = auto
testpaths = tests
addopts = --cov=app --cov-report=term-missing --cov-fail-under=80
```

## Go (testing package)

```go
// user_service_test.go
package service_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "github.com/stretchr/testify/require"
)

// Table-driven test
func TestCreateUser(t *testing.T) {
    tests := []struct {
        name      string
        input     CreateUserInput
        mockUser  *User
        mockErr   error
        wantErr   bool
    }{
        {
            name:     "valid input creates user",
            input:    CreateUserInput{Email: "a@b.com", Name: "Alice"},
            mockUser: &User{ID: "1", Email: "a@b.com"},
            wantErr:  false,
        },
        {
            name:    "duplicate email returns error",
            input:   CreateUserInput{Email: "a@b.com", Name: "Alice"},
            mockErr: ErrDuplicateEmail,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := new(MockUserRepo)
            repo.On("Insert", mock.Anything, tt.input).Return(tt.mockUser, tt.mockErr)

            svc := NewUserService(repo)
            user, err := svc.CreateUser(context.Background(), tt.input)

            if tt.wantErr {
                require.Error(t, err)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.mockUser, user)
            repo.AssertExpectations(t)
        })
    }
}

// Skip in short mode
func TestExpensiveOperation(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping in short mode")
    }
    // ...
}

// t.Helper() — marks as helper so failures point to caller
func assertUserEqual(t *testing.T, want, got *User) {
    t.Helper()
    assert.Equal(t, want.ID, got.ID)
    assert.Equal(t, want.Email, got.Email)
}
```

```bash
# Run tests with race detector
go test -race ./...

# Short tests only (skip integration/slow)
go test -short ./...

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## JUnit 5 (Java / Kotlin)

```java
import org.junit.jupiter.api.*;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.*;
import org.mockito.*;
import static org.mockito.Mockito.*;
import static org.assertj.core.api.Assertions.*;

@ExtendWith(MockitoExtension.class)
class UserServiceTest {

    @Mock
    UserRepository userRepository;

    @InjectMocks
    UserService userService;

    @Test
    void createUser_returnsUser_whenValidInput() {
        var user = new User("1", "a@b.com", "Alice");
        when(userRepository.save(any())).thenReturn(user);

        var result = userService.createUser("a@b.com", "Alice");

        assertThat(result).isEqualTo(user);
        verify(userRepository).save(any(User.class));
    }

    @ParameterizedTest
    @CsvSource({"'', Alice", "not-email, Alice", "a@b.com, ''"})
    void createUser_throwsValidationException_forInvalidInput(String email, String name) {
        assertThatThrownBy(() -> userService.createUser(email, name))
            .isInstanceOf(ValidationException.class);
    }

    @ParameterizedTest
    @MethodSource("provideUserInputs")
    void createUser_setsCorrectFields(String email, String name, String expected) {
        // ...
    }

    static Stream<Arguments> provideUserInputs() {
        return Stream.of(
            Arguments.of("a@b.com", "Alice", "alice"),
            Arguments.of("B@C.COM", "Bob", "bob")
        );
    }

    // Argument captor
    @Test
    void createUser_savesWithHashedPassword() {
        var captor = ArgumentCaptor.forClass(User.class);
        userService.createUser("a@b.com", "Alice");
        verify(userRepository).save(captor.capture());
        assertThat(captor.getValue().getPasswordHash()).isNotBlank();
    }
}
```

## xUnit (.NET)

```csharp
using Xunit;
using Moq;
using FluentAssertions;

public class UserServiceTests
{
    private readonly Mock<IUserRepository> _mockRepo;
    private readonly UserService _service;

    public UserServiceTests()
    {
        _mockRepo = new Mock<IUserRepository>();
        _service = new UserService(_mockRepo.Object);
    }

    [Fact]
    public async Task CreateUser_ReturnsUser_WhenValidInput()
    {
        var user = new User { Id = "1", Email = "a@b.com" };
        _mockRepo.Setup(r => r.InsertAsync(It.IsAny<User>())).ReturnsAsync(user);

        var result = await _service.CreateUserAsync("a@b.com", "Alice");

        result.Should().BeEquivalentTo(user);
        _mockRepo.Verify(r => r.InsertAsync(It.IsAny<User>()), Times.Once);
    }

    [Theory]
    [InlineData("", "Alice")]
    [InlineData("not-email", "Alice")]
    [InlineData("a@b.com", "")]
    public async Task CreateUser_ThrowsValidationException_ForInvalidInput(string email, string name)
    {
        await Assert.ThrowsAsync<ValidationException>(
            () => _service.CreateUserAsync(email, name));
    }
}
```

## Key Rules

- Name tests: `<Unit>_<Scenario>_<ExpectedBehavior>`.
- One logical assertion per test — multiple `assert` calls are fine if they verify the same outcome.
- Mock at the boundary closest to the unit under test; avoid mocking internal collaborators.
- Use `t.Helper()` (Go) / helper fixtures to avoid duplicating setup.
- Never share mutable state between test cases — always reset mocks in `beforeEach` / `setUp`.
- Use `--race` flag in Go CI to catch data races.
- Maintain ≥ 80% line coverage; prioritize branch coverage for business logic.
