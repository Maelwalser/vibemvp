# Testing: Frontend Unit & Component Tests Skill Guide

## Overview

Vitest with vite config integration, Testing Library queries and interactions, Storybook CSF3 stories with play functions, and MSW for API mocking.

## Vitest Configuration

```typescript
// vite.config.ts
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    globals: true,                // no need to import describe/it/expect
    setupFiles: ['./src/test/setup.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html', 'lcov'],
      thresholds: {
        lines: 80,
        functions: 80,
        branches: 80,
      },
      exclude: ['src/test/**', '**/*.stories.*', 'src/main.tsx'],
    },
  },
});
```

```typescript
// src/test/setup.ts
import '@testing-library/jest-dom';
import { cleanup } from '@testing-library/react';
import { afterEach, vi } from 'vitest';

afterEach(() => {
  cleanup();
});

// Mock global browser APIs not in jsdom
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
  })),
});
```

## Module Mocking

```typescript
// Mock entire module
vi.mock('../api/users', () => ({
  fetchUsers: vi.fn().mockResolvedValue([
    { id: '1', email: 'alice@example.com', name: 'Alice' },
  ]),
  createUser: vi.fn().mockResolvedValue({ id: '2', email: 'bob@example.com' }),
}));

// Mock with factory (for ES modules with default export)
vi.mock('../hooks/useAuth', () => ({
  default: vi.fn().mockReturnValue({
    user: { id: '1', email: 'alice@example.com' },
    isAuthenticated: true,
    logout: vi.fn(),
  }),
}));

// Partial mock — keep real implementations, override some
vi.mock('../utils/date', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../utils/date')>();
  return {
    ...actual,
    formatDate: vi.fn().mockReturnValue('Jan 15, 2024'),
  };
});
```

## Testing Library

### Preferred Query Priority

```typescript
// 1. BEST: Role-based (accessible)
screen.getByRole('button', { name: 'Submit' });
screen.getByRole('heading', { name: 'User Profile' });
screen.getByRole('textbox', { name: 'Email' });
screen.getByRole('combobox', { name: 'Country' });

// 2. Label-based (accessible)
screen.getByLabelText('Email address');

// 3. Text content
screen.getByText('Welcome, Alice');
screen.getByText(/welcome/i);  // regex, case-insensitive

// 4. Placeholder
screen.getByPlaceholderText('Enter your email');

// 5. Test ID (last resort)
screen.getByTestId('user-avatar');
```

### Component Render Test

```typescript
// UserCard.test.tsx
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { UserCard } from './UserCard';

describe('UserCard', () => {
  const user = { id: '1', name: 'Alice', email: 'alice@example.com', role: 'admin' };

  it('renders user name and email', () => {
    render(<UserCard user={user} />);

    expect(screen.getByRole('heading', { name: 'Alice' })).toBeInTheDocument();
    expect(screen.getByText('alice@example.com')).toBeInTheDocument();
  });

  it('calls onEdit when edit button is clicked', async () => {
    const onEdit = vi.fn();
    render(<UserCard user={user} onEdit={onEdit} />);

    await userEvent.click(screen.getByRole('button', { name: 'Edit user' }));

    expect(onEdit).toHaveBeenCalledWith(user);
    expect(onEdit).toHaveBeenCalledTimes(1);
  });

  it('shows admin badge for admin users', () => {
    render(<UserCard user={user} />);
    expect(screen.getByText('Admin')).toBeInTheDocument();
  });

  it('hides edit button when readonly prop is set', () => {
    render(<UserCard user={user} readonly />);
    expect(screen.queryByRole('button', { name: 'Edit user' })).not.toBeInTheDocument();
  });
});
```

### User Events

```typescript
import userEvent from '@testing-library/user-event';

// Setup (v14+ API)
const user = userEvent.setup();

await user.click(screen.getByRole('button'));
await user.type(screen.getByLabelText('Email'), 'alice@example.com');
await user.clear(screen.getByLabelText('Email'));
await user.selectOptions(screen.getByRole('combobox'), 'Canada');
await user.keyboard('{Enter}');
await user.tab();
await user.upload(screen.getByLabelText('File'), new File(['content'], 'test.txt'));
```

### Async Assertions

```typescript
import { waitFor, waitForElementToBeRemoved } from '@testing-library/react';

// Wait for element to appear
await waitFor(() => {
  expect(screen.getByText('Data loaded')).toBeInTheDocument();
});

// Wait for element to disappear
await waitForElementToBeRemoved(() => screen.queryByRole('progressbar'));

// Find* queries — built-in async
const element = await screen.findByText('Welcome, Alice');
```

### renderHook for Custom Hooks

```typescript
import { renderHook, act } from '@testing-library/react';
import { useCounter } from './useCounter';

describe('useCounter', () => {
  it('increments counter', () => {
    const { result } = renderHook(() => useCounter(0));

    act(() => {
      result.current.increment();
    });

    expect(result.current.count).toBe(1);
  });

  it('uses query context', () => {
    const wrapper = ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );

    const { result } = renderHook(() => useUsers(), { wrapper });
    // ...
  });
});
```

## Storybook

### CSF3 Story Format

```typescript
// UserCard.stories.tsx
import type { Meta, StoryObj } from '@storybook/react';
import { within, userEvent, expect } from '@storybook/test';
import { UserCard } from './UserCard';

const meta: Meta<typeof UserCard> = {
  title: 'Components/UserCard',
  component: UserCard,
  tags: ['autodocs'],
  argTypes: {
    onEdit: { action: 'onEdit' },
  },
};
export default meta;

type Story = StoryObj<typeof UserCard>;

export const Default: Story = {
  args: {
    user: { id: '1', name: 'Alice', email: 'alice@example.com', role: 'user' },
  },
};

export const AdminUser: Story = {
  args: {
    user: { id: '1', name: 'Alice', email: 'alice@example.com', role: 'admin' },
  },
};

// Interaction test with play function
export const EditInteraction: Story = {
  args: {
    user: { id: '1', name: 'Alice', email: 'alice@example.com', role: 'user' },
  },
  play: async ({ canvasElement, args }) => {
    const canvas = within(canvasElement);

    await userEvent.click(canvas.getByRole('button', { name: 'Edit user' }));

    await expect(args.onEdit).toHaveBeenCalledWith(
      expect.objectContaining({ id: '1' })
    );
  },
};

// Loading state
export const Loading: Story = {
  args: { user: undefined, isLoading: true },
};
```

### MSW Integration (Storybook API Mocking)

```typescript
// .storybook/preview.tsx
import { initialize, mswLoader } from 'msw-storybook-addon';

initialize();

export const loaders = [mswLoader];
```

```typescript
// UserList.stories.tsx
import { http, HttpResponse } from 'msw';

export const WithUsers: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get('/api/users', () =>
          HttpResponse.json([
            { id: '1', name: 'Alice', email: 'alice@example.com' },
            { id: '2', name: 'Bob', email: 'bob@example.com' },
          ])
        ),
      ],
    },
  },
};

export const EmptyState: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get('/api/users', () => HttpResponse.json([])),
      ],
    },
  },
};

export const ErrorState: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get('/api/users', () =>
          HttpResponse.json({ error: 'Internal Server Error' }, { status: 500 })
        ),
      ],
    },
  },
};
```

### Accessibility Addon

```bash
# Install
npm install -D @storybook/addon-a11y

# .storybook/main.ts
addons: ['@storybook/addon-a11y']
```

```typescript
// Disable specific a11y rule per story
export const SpecialCase: Story = {
  parameters: {
    a11y: {
      config: {
        rules: [{ id: 'color-contrast', enabled: false }],
      },
    },
  },
};
```

## Key Rules

- Use `getByRole` first — it tests accessibility and semantics simultaneously.
- Never use `getByTestId` for assertions that can be expressed as role/label/text queries.
- Use `userEvent` over `fireEvent` — it simulates real browser events including focus, keyboard, pointer.
- Mock at the network layer with MSW in Storybook — not at the import level.
- Add `play` functions to Storybook stories for any interactive component — they double as interaction tests.
- Run `vitest --coverage` in CI and fail if thresholds are not met.
- Keep stories in the same directory as components — not in a separate `stories/` folder.
