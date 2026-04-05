# Web Linting Skill Guide

## ESLint Flat Config (eslint.config.js)

### TypeScript + React Setup

```ts
// eslint.config.js
import { defineConfig } from 'eslint/config'
import js from '@eslint/js'
import tsPlugin from '@typescript-eslint/eslint-plugin'
import tsParser from '@typescript-eslint/parser'
import reactHooksPlugin from 'eslint-plugin-react-hooks'
import prettierConfig from 'eslint-config-prettier'

export default defineConfig([
  // Base JS rules
  { files: ['**/*.{js,mjs,cjs}'], ...js.configs.recommended },

  // TypeScript files
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        project: './tsconfig.json',
        tsconfigRootDir: import.meta.dirname,
      },
    },
    plugins: {
      '@typescript-eslint': tsPlugin,
      'react-hooks': reactHooksPlugin,
    },
    rules: {
      ...tsPlugin.configs.recommended.rules,
      ...tsPlugin.configs['recommended-type-checked'].rules,
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'warn',

      // Custom overrides
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      '@typescript-eslint/explicit-function-return-type': 'off',
      '@typescript-eslint/no-explicit-any': 'warn',
      '@typescript-eslint/no-floating-promises': 'error',
      '@typescript-eslint/no-misused-promises': 'error',
      'no-console': ['warn', { allow: ['warn', 'error'] }],
    },
  },

  // Disable rules that conflict with Prettier (always last)
  prettierConfig,

  // Ignore patterns
  {
    ignores: ['dist/**', '.next/**', 'node_modules/**', '*.min.js'],
  },
])
```

### package.json Scripts

```json
{
  "scripts": {
    "lint": "eslint . --max-warnings 0",
    "lint:fix": "eslint . --fix",
    "type-check": "tsc --noEmit"
  }
}
```

---

## Biome (Unified Linter + Formatter)

Biome replaces ESLint + Prettier in a single fast Rust tool.

```json
// biome.json
{
  "$schema": "https://biomejs.dev/schemas/1.9.0/schema.json",
  "vcs": { "enabled": true, "clientKind": "git", "useIgnoreFile": true },
  "files": {
    "ignoreUnknown": false,
    "ignore": ["dist", ".next", "node_modules"]
  },
  "formatter": {
    "enabled": true,
    "indentStyle": "space",
    "indentWidth": 2,
    "lineWidth": 100
  },
  "linter": {
    "enabled": true,
    "rules": {
      "recommended": true,
      "correctness": {
        "noUnusedVariables": "error",
        "useExhaustiveDependencies": "warn"
      },
      "suspicious": {
        "noExplicitAny": "warn"
      },
      "style": {
        "noNonNullAssertion": "warn",
        "useConst": "error"
      },
      "security": {
        "noDangerouslySetInnerHtmlWithChildren": "error"
      }
    }
  },
  "javascript": {
    "formatter": {
      "quoteStyle": "single",
      "semicolons": "asNeeded",
      "trailingCommas": "es5"
    }
  },
  "organizeImports": { "enabled": true }
}
```

```json
// package.json
{
  "scripts": {
    "lint": "biome check .",
    "lint:fix": "biome check --write .",
    "format": "biome format --write ."
  }
}
```

---

## oxlint (Fast Pre-Check)

oxlint is a Rust-based linter — run it as a fast first pass before ESLint.

```json
// package.json
{
  "scripts": {
    "lint": "oxlint . && eslint . --max-warnings 0"
  }
}
```

```json
// .oxlintrc.json
{
  "rules": {
    "no-unused-vars": "error",
    "no-undef": "error",
    "eqeqeq": "error"
  },
  "ignorePatterns": ["dist/", ".next/", "node_modules/"]
}
```

---

## Stylelint (CSS / SCSS)

```js
// .stylelintrc.js
module.exports = {
  extends: [
    'stylelint-config-standard',
    'stylelint-config-standard-scss',   // if using SCSS
    'stylelint-config-prettier',         // disable rules conflicting with Prettier
  ],
  plugins: ['stylelint-order'],
  rules: {
    'order/properties-alphabetical-order': true,
    'color-no-invalid-hex': true,
    'declaration-no-important': true,
    'selector-class-pattern': '^[a-z][a-z0-9-]*$',   // kebab-case class names
    'scss/at-rule-no-unknown': [true, {
      ignoreAtRules: ['tailwind', 'apply', 'layer', 'config'],
    }],
  },
}
```

```json
// package.json
{
  "scripts": {
    "lint:css": "stylelint '**/*.{css,scss}' --fix"
  }
}
```

---

## lint-staged (Run on Staged Files Only)

```json
// package.json
{
  "lint-staged": {
    "*.{ts,tsx,js,jsx}": [
      "eslint --fix --max-warnings 0",
      "prettier --write"
    ],
    "*.{css,scss}": [
      "stylelint --fix",
      "prettier --write"
    ],
    "*.{json,md,yaml,yml}": "prettier --write"
  }
}
```

```sh
# Setup with Husky
npx husky init
echo "npx lint-staged" > .husky/pre-commit
```

### With Biome (simpler)

```json
// package.json
{
  "lint-staged": {
    "*.{ts,tsx,js,jsx,json,css}": "biome check --write"
  }
}
```

---

## CI: GitHub Actions

```yaml
# .github/workflows/lint.yml
name: Lint & Type Check

on:
  pull_request:
  push:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: pnpm

      - run: pnpm install --frozen-lockfile

      - name: Type check
        run: pnpm type-check

      - name: Lint
        run: pnpm lint

      - name: Format check
        run: pnpm prettier --check .
```

### With Biome in CI

```yaml
      - name: Biome check
        run: npx @biomejs/biome ci .
        # exits non-zero on any lint/format issue — no --write in CI
```

---

## Prettier Config

```json
// .prettierrc
{
  "semi": false,
  "singleQuote": true,
  "tabWidth": 2,
  "trailingComma": "es5",
  "printWidth": 100,
  "arrowParens": "always",
  "endOfLine": "lf",
  "plugins": ["prettier-plugin-tailwindcss"]
}
```

```json
// .prettierignore
dist/
.next/
node_modules/
*.min.js
pnpm-lock.yaml
```

---

## Key Rules

- ESLint flat config (`eslint.config.js`) is the modern format — avoid legacy `.eslintrc.*` for new projects.
- Always put `eslint-config-prettier` last in the config array to disable conflicting formatting rules.
- Use Biome when you want a single fast tool for both linting and formatting — it's not yet 1:1 with all ESLint plugins.
- Run oxlint before ESLint in CI for faster feedback on common errors.
- lint-staged ensures only staged files are checked — never run `eslint .` on every file in a pre-commit hook.
- `--max-warnings 0` in CI turns all warnings into errors — prevents warning accumulation.
- Never skip hooks with `--no-verify` — fix the underlying lint issue instead.
