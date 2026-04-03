# Testing: Playwright & Cypress Skill Guide

## Overview

Playwright browser automation and Cypress E2E testing — page interactions, API mocking, visual comparison, trace viewer, and component testing.

## Playwright

### Configuration

```typescript
// playwright.config.ts
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [['html', { open: 'never' }], ['junit', { outputFile: 'results.xml' }]],
  use: {
    baseURL: process.env.BASE_URL || 'http://localhost:3000',
    trace: 'on-first-retry',      // capture trace on retry
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
    { name: 'firefox',  use: { ...devices['Desktop Firefox'] } },
    { name: 'webkit',   use: { ...devices['Desktop Safari'] } },
    { name: 'Mobile Chrome', use: { ...devices['Pixel 5'] } },
  ],
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
  },
});
```

### Page Interactions

```typescript
import { test, expect, Page } from '@playwright/test';

test.describe('User registration', () => {
  test('registers a new user successfully', async ({ page }) => {
    await page.goto('/register');

    await page.getByLabel('Email').fill('alice@example.com');
    await page.getByLabel('Password').fill('Secure123!');
    await page.getByLabel('Confirm Password').fill('Secure123!');
    await page.getByRole('button', { name: 'Create account' }).click();

    await expect(page.getByRole('heading', { name: 'Welcome, Alice' })).toBeVisible();
    await expect(page).toHaveURL('/dashboard');
  });

  test('shows error for duplicate email', async ({ page }) => {
    await page.goto('/register');
    await page.getByLabel('Email').fill('existing@example.com');
    await page.getByRole('button', { name: 'Create account' }).click();

    await expect(page.getByText('Email already in use')).toBeVisible();
  });
});
```

### Assertions

```typescript
// Visibility
await expect(locator).toBeVisible();
await expect(locator).toBeHidden();

// Text content
await expect(locator).toHaveText('exact match');
await expect(locator).toContainText('partial');

// Input value
await expect(locator).toHaveValue('input value');

// URL
await expect(page).toHaveURL('/dashboard');
await expect(page).toHaveURL(/dashboard/);

// Count
await expect(page.getByRole('listitem')).toHaveCount(5);

// Attribute
await expect(locator).toHaveAttribute('aria-disabled', 'true');
```

### API Mocking with page.route

```typescript
test('shows fallback when API fails', async ({ page }) => {
  // Mock specific endpoint
  await page.route('**/api/users', route => {
    route.fulfill({
      status: 500,
      contentType: 'application/json',
      body: JSON.stringify({ error: 'internal server error' }),
    });
  });

  await page.goto('/users');
  await expect(page.getByText('Failed to load users')).toBeVisible();
});

test('intercepts and modifies response', async ({ page }) => {
  await page.route('**/api/products', async route => {
    const response = await route.fetch();
    const json = await response.json();
    json.push({ id: 'mock-1', name: 'Mocked Product' });
    route.fulfill({ response, json });
  });
});
```

### Visual Comparison

```typescript
// Screenshot comparison (pixel-level)
await expect(page).toHaveScreenshot('homepage.png');

// Element screenshot
await expect(page.getByTestId('chart')).toHaveScreenshot('chart.png', {
  maxDiffPixelRatio: 0.01,  // allow 1% pixel difference
});

// Update baselines: npx playwright test --update-snapshots
```

### Fixtures & Reusable Auth

```typescript
// fixtures.ts
import { test as base, Page } from '@playwright/test';

type MyFixtures = {
  authenticatedPage: Page;
};

export const test = base.extend<MyFixtures>({
  authenticatedPage: async ({ page }, use) => {
    await page.goto('/login');
    await page.getByLabel('Email').fill('test@example.com');
    await page.getByLabel('Password').fill('password');
    await page.getByRole('button', { name: 'Sign in' }).click();
    await page.waitForURL('/dashboard');
    await use(page);
  },
});

// test file
import { test } from './fixtures';

test('authenticated user can view profile', async ({ authenticatedPage: page }) => {
  await page.goto('/profile');
  await expect(page.getByText('test@example.com')).toBeVisible();
});
```

### Trace Viewer

```bash
# Run with trace always
npx playwright test --trace on

# View trace
npx playwright show-trace trace.zip
```

## Cypress

### Configuration

```typescript
// cypress.config.ts
import { defineConfig } from 'cypress';

export default defineConfig({
  e2e: {
    baseUrl: 'http://localhost:3000',
    specPattern: 'cypress/e2e/**/*.cy.ts',
    supportFile: 'cypress/support/e2e.ts',
    setupNodeEvents(on, config) {
      on('task', {
        // DB operations
        async resetDatabase() {
          await db.query('TRUNCATE TABLE users CASCADE');
          return null;
        },
        async seedUser(user) {
          return db.query('INSERT INTO users ...', user);
        },
      });
    },
  },
  component: {
    devServer: { framework: 'react', bundler: 'vite' },
  },
});
```

### Page Interactions

```typescript
// cypress/e2e/register.cy.ts
describe('User registration', () => {
  beforeEach(() => {
    cy.task('resetDatabase');
    cy.visit('/register');
  });

  it('registers a new user', () => {
    cy.get('[data-cy="email"]').type('alice@example.com');
    cy.get('[data-cy="password"]').type('Secure123!');
    cy.get('[data-cy="submit"]').click();

    cy.url().should('include', '/dashboard');
    cy.contains('Welcome, Alice').should('be.visible');
  });
});
```

### Intercept (API Mocking)

```typescript
// Stub API response
cy.intercept('GET', '/api/users', { fixture: 'users.json' }).as('getUsers');

cy.visit('/users');
cy.wait('@getUsers');

cy.get('[data-cy="user-list"]').should('have.length', 3);
```

```typescript
// Intercept and modify
cy.intercept('POST', '/api/orders', (req) => {
  req.reply({
    statusCode: 201,
    body: { id: 'order-123', status: 'created' },
  });
}).as('createOrder');

cy.get('[data-cy="checkout"]').click();
cy.wait('@createOrder').its('response.statusCode').should('eq', 201);
```

### Session for Auth Reuse

```typescript
// cypress/support/commands.ts
Cypress.Commands.add('login', (email = 'test@example.com', password = 'password') => {
  cy.session([email, password], () => {
    cy.request('POST', '/api/auth/login', { email, password })
      .its('body.token')
      .then(token => {
        window.localStorage.setItem('auth_token', token);
      });
  });
});

// Usage in tests
beforeEach(() => {
  cy.login();
  cy.visit('/dashboard');
});
```

### Component Testing

```typescript
// cypress/component/UserCard.cy.tsx
import { UserCard } from '../../src/components/UserCard';

describe('UserCard', () => {
  it('renders user name and email', () => {
    cy.mount(<UserCard user={{ id: '1', name: 'Alice', email: 'a@b.com' }} />);

    cy.contains('Alice').should('be.visible');
    cy.contains('a@b.com').should('be.visible');
  });

  it('calls onEdit when edit button clicked', () => {
    const onEdit = cy.stub().as('onEdit');
    cy.mount(<UserCard user={{ id: '1', name: 'Alice', email: 'a@b.com' }} onEdit={onEdit} />);

    cy.get('[data-cy="edit-btn"]').click();
    cy.get('@onEdit').should('have.been.calledOnce');
  });
});
```

## Key Rules

- Use `getByRole`, `getByLabel`, `getByText` (Playwright) / `data-cy` attributes (Cypress) — never `getByTestId` or CSS selectors for user-facing assertions.
- Always `await expect(locator).toBeVisible()` before interacting — Playwright auto-waits, but explicit assertion documents intent.
- Mock external APIs in tests — never call real third-party services.
- Use `cy.session` / Playwright auth fixtures to avoid logging in on every test.
- Keep test isolation: reset DB state in `beforeEach` / `beforeAll`.
- Store visual snapshots in version control; run `--update-snapshots` only intentionally.
- In CI, set `retries: 2` for flaky network conditions; investigate and fix persistent failures.
