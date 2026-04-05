# UI Styling Skill Guide

## Tailwind CSS

### Utility Class Patterns

```html
<!-- Spacing, sizing, layout -->
<div class="flex items-center gap-4 p-6 rounded-xl shadow-md bg-white">
  <span class="text-sm font-semibold text-gray-700 truncate">Label</span>
</div>

<!-- Border and background with opacity -->
<div class="border border-gray-200 bg-gray-50/80 backdrop-blur-sm">
```

### Responsive Breakpoints

```html
<!-- Mobile-first: base → sm(640px) → md(768px) → lg(1024px) → xl(1280px) -->
<div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
  <div class="text-sm md:text-base lg:text-lg">Responsive text</div>
  <div class="hidden md:block">Desktop only</div>
  <div class="block md:hidden">Mobile only</div>
</div>
```

### Dark Mode

```html
<!-- dark: prefix — toggled by class="dark" on <html> or prefers-color-scheme -->
<div class="bg-white dark:bg-gray-900 text-gray-900 dark:text-gray-100">
  <button class="bg-blue-500 hover:bg-blue-600 dark:bg-blue-600 dark:hover:bg-blue-700">
    Button
  </button>
</div>
```

### Group Hover

```html
<!-- group on parent, group-hover: on children -->
<div class="group flex items-center gap-2 cursor-pointer">
  <span class="text-gray-600 group-hover:text-blue-600 transition-colors">Link</span>
  <svg class="opacity-0 group-hover:opacity-100 transition-opacity" />
</div>
```

### @layer Components for Custom Classes

```css
/* globals.css */
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer components {
  .btn-primary {
    @apply px-4 py-2 bg-blue-500 text-white rounded-lg font-medium
           hover:bg-blue-600 active:scale-95 transition-all;
  }

  .card {
    @apply bg-white dark:bg-gray-800 rounded-xl shadow-sm border
           border-gray-200 dark:border-gray-700 p-6;
  }

  .input-field {
    @apply w-full px-3 py-2 border border-gray-300 dark:border-gray-600
           rounded-lg bg-white dark:bg-gray-900 focus:ring-2
           focus:ring-blue-500 focus:border-transparent outline-none;
  }
}
```

### tailwind.config.js Theme Extension

```js
/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['./src/**/*.{ts,tsx,html}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        brand: {
          50:  '#eff6ff',
          500: '#3b82f6',
          900: '#1e3a8a',
        },
      },
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui'],
        mono: ['JetBrains Mono', 'ui-monospace'],
      },
      spacing: {
        18: '4.5rem',
        22: '5.5rem',
      },
      borderRadius: {
        '4xl': '2rem',
      },
      animation: {
        'fade-in': 'fadeIn 0.2s ease-in-out',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0', transform: 'translateY(4px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
      },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
    require('@tailwindcss/typography'),
  ],
}
```

---

## CSS Modules

```tsx
// Button.module.css
.button {
  padding: 0.5rem 1rem;
  background: var(--color-primary);
  border-radius: 0.375rem;
  font-weight: 600;
}

.button:hover {
  background: var(--color-primary-dark);
}

.buttonLarge {
  composes: button;
  padding: 0.75rem 1.5rem;
  font-size: 1.125rem;
}
```

```tsx
// Button.tsx
import styles from './Button.module.css'

export function Button({ large }: { large?: boolean }) {
  return (
    <button className={large ? styles.buttonLarge : styles.button}>
      Click me
    </button>
  )
}

// Combining with cn/clsx
import clsx from 'clsx'
<div className={clsx(styles.card, isActive && styles.cardActive)} />
```

---

## Styled Components / Emotion

```tsx
import styled, { css } from 'styled-components'

// Basic styled component
const Button = styled.button<{ variant?: 'primary' | 'ghost' }>`
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  font-weight: 600;
  transition: all 0.15s ease;

  ${({ variant = 'primary' }) =>
    variant === 'primary'
      ? css`
          background: ${({ theme }) => theme.colors.primary};
          color: white;
          &:hover { background: ${({ theme }) => theme.colors.primaryDark}; }
        `
      : css`
          background: transparent;
          border: 1px solid currentColor;
          &:hover { background: rgba(0,0,0,0.05); }
        `}
`

// Extending styles
const LargeButton = styled(Button)`
  padding: 0.75rem 1.5rem;
  font-size: 1.125rem;
`

// ThemeProvider
import { ThemeProvider } from 'styled-components'

const theme = {
  colors: { primary: '#3b82f6', primaryDark: '#2563eb' },
  spacing: { sm: '0.5rem', md: '1rem', lg: '1.5rem' },
}

function App() {
  return (
    <ThemeProvider theme={theme}>
      <Button variant="primary">Submit</Button>
    </ThemeProvider>
  )
}
```

### Emotion (css prop)

```tsx
/** @jsxImportSource @emotion/react */
import { css } from '@emotion/react'

const cardStyle = css`
  padding: 1.5rem;
  border-radius: 0.75rem;
  box-shadow: 0 1px 3px rgba(0,0,0,0.1);
`

function Card({ children }: { children: React.ReactNode }) {
  return <div css={cardStyle}>{children}</div>
}
```

---

## Sass / SCSS

```scss
// variables and nesting
$primary: #3b82f6;
$primary-dark: darken($primary, 10%);
$border-radius: 0.375rem;

.card {
  padding: 1.5rem;
  border-radius: $border-radius;

  &__header {
    font-size: 1.125rem;
    font-weight: 600;
    margin-bottom: 1rem;
  }

  &__body {
    color: #6b7280;
  }

  &--highlighted {
    border: 2px solid $primary;
  }
}

// Mixins
@mixin flex-center {
  display: flex;
  align-items: center;
  justify-content: center;
}

@mixin responsive($breakpoint) {
  @if $breakpoint == 'md' {
    @media (min-width: 768px) { @content; }
  } @else if $breakpoint == 'lg' {
    @media (min-width: 1024px) { @content; }
  }
}

.hero {
  @include flex-center;
  height: 100vh;

  @include responsive('md') {
    height: 80vh;
  }
}
```

---

## UnoCSS

```ts
// uno.config.ts
import { defineConfig, presetUno, presetAttributify } from 'unocss'

export default defineConfig({
  presets: [
    presetUno(),          // Tailwind/Windi CSS compatible utilities
    presetAttributify(),  // <div flex items-center />
  ],
  shortcuts: {
    'btn': 'px-4 py-2 rounded-lg font-medium transition-colors',
    'btn-primary': 'btn bg-blue-500 text-white hover:bg-blue-600',
  },
  rules: [
    // Custom rule: text-shadow-sm
    ['text-shadow-sm', { 'text-shadow': '0 1px 2px rgba(0,0,0,0.1)' }],
    // Dynamic rule: gradient-from-[color]
    [/^gradient-from-(.+)$/, ([, color]) => ({
      'background-image': `linear-gradient(to right, ${color}, transparent)`,
    })],
  ],
  theme: {
    colors: {
      brand: '#3b82f6',
    },
  },
})
```

```html
<!-- Attributify mode -->
<div flex items-center gap-4 p-6 bg-white rounded-xl>
  <span text-sm font-semibold text-gray-700>Label</span>
</div>
```

---

## CSS Custom Properties

```css
/* :root for global tokens */
:root {
  --color-primary: #3b82f6;
  --color-primary-dark: #2563eb;
  --spacing-sm: 0.5rem;
  --spacing-md: 1rem;
  --radius-md: 0.375rem;
  --font-sans: 'Inter', system-ui, sans-serif;
}

/* Dark mode override */
@media (prefers-color-scheme: dark) {
  :root {
    --color-bg: #111827;
    --color-text: #f9fafb;
    --color-border: #374151;
  }
}

/* Class-based dark mode */
.dark {
  --color-bg: #111827;
  --color-text: #f9fafb;
}

/* Responsive tokens */
:root {
  --container-width: 100%;
}
@media (min-width: 768px) {
  :root { --container-width: 768px; }
}
@media (min-width: 1024px) {
  :root { --container-width: 1024px; }
}

/* Usage */
.card {
  background: var(--color-bg, white);
  color: var(--color-text, #111827);
  border-radius: var(--radius-md);
  padding: var(--spacing-md);
}
```

---

## Key Rules

- Use Tailwind for utility-first projects — avoid inline styles.
- CSS Modules for component-scoped styles without runtime cost.
- Styled Components/Emotion when you need dynamic styles driven by props/theme.
- CSS custom properties for design tokens shared across CSS-in-JS and plain CSS.
- All color values go in config/theme files — never hardcode hex values inline.
