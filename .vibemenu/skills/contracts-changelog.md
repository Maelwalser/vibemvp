# Contracts: Changelog & Conventional Commits Skill Guide

## Overview

Conventional Commits format, commitlint, Husky pre-commit hooks, semantic-release automated changelog, and CHANGELOG.md format.

## Conventional Commits Format

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Types

| Type | When to Use | SemVer bump |
|------|-------------|-------------|
| `feat` | New feature | Minor |
| `fix` | Bug fix | Patch |
| `refactor` | Code change that is not feat/fix | None |
| `perf` | Performance improvement | Patch |
| `docs` | Documentation only | None |
| `test` | Adding or fixing tests | None |
| `chore` | Build process, dependency updates | None |
| `ci` | CI/CD configuration | None |
| `build` | Build system changes | None |
| `revert` | Reverts a previous commit | Depends |

### Breaking Change → Major Bump

```
feat(auth)!: replace JWT with session-based auth

BREAKING CHANGE: The Authorization header is no longer accepted.
Clients must now use the Set-Cookie flow. See migration guide at
https://docs.example.com/migration/auth-v2.
```

### Examples

```
feat(users): add role-based access control
fix(orders): prevent duplicate order creation on retry
refactor(db): extract connection pool to shared module
perf(cache): add Redis caching for user lookups
docs(api): update OpenAPI spec for v2 endpoints
test(users): add integration tests for createUser endpoint
chore(deps): bump express from 4.18.0 to 4.19.0
ci(github): add matrix build for Node 18 and 20
```

## commitlint Configuration

```javascript
// .commitlintrc.js
module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    'type-enum': [
      2,  // level: error
      'always',
      ['feat', 'fix', 'refactor', 'perf', 'docs', 'test', 'chore', 'ci', 'build', 'revert'],
    ],
    'subject-case': [2, 'always', 'lower-case'],
    'subject-max-length': [2, 'always', 100],
    'body-max-line-length': [1, 'always', 120],  // warning only
    'scope-case': [2, 'always', 'lower-case'],
  },
};
```

```bash
# Install
npm install -D @commitlint/cli @commitlint/config-conventional
```

## Husky Pre-Commit & Commit-msg Hooks

```bash
# Install Husky
npm install -D husky lint-staged
npx husky init

# Add commit-msg hook (validates commit message with commitlint)
echo 'npx --no -- commitlint --edit $1' > .husky/commit-msg
chmod +x .husky/commit-msg

# Add pre-commit hook (lint + format staged files)
cat > .husky/pre-commit << 'EOF'
npx lint-staged
EOF
chmod +x .husky/pre-commit
```

```json
// package.json — lint-staged config
{
  "lint-staged": {
    "*.{ts,tsx,js,jsx}": ["eslint --fix", "prettier --write"],
    "*.{json,yaml,yml,md}": ["prettier --write"],
    "*.go": ["gofmt -w", "go vet"]
  }
}
```

```json
// package.json — husky prepare script
{
  "scripts": {
    "prepare": "husky"
  }
}
```

## semantic-release

### Pipeline: analyzeCommits → generateNotes → createRelease → publish

```json
// .releaserc.json
{
  "branches": ["main", { "name": "next", "prerelease": true }],
  "plugins": [
    "@semantic-release/commit-analyzer",
    "@semantic-release/release-notes-generator",
    ["@semantic-release/changelog", {
      "changelogFile": "CHANGELOG.md"
    }],
    ["@semantic-release/npm", {
      "npmPublish": true
    }],
    ["@semantic-release/github", {
      "assets": ["dist/*.js", "dist/*.d.ts"]
    }],
    ["@semantic-release/git", {
      "assets": ["CHANGELOG.md", "package.json"],
      "message": "chore(release): ${nextRelease.version} [skip ci]\n\n${nextRelease.notes}"
    }]
  ]
}
```

```bash
# Install
npm install -D semantic-release \
  @semantic-release/commit-analyzer \
  @semantic-release/release-notes-generator \
  @semantic-release/changelog \
  @semantic-release/github \
  @semantic-release/npm \
  @semantic-release/git
```

### GitHub Actions Workflow

```yaml
# .github/workflows/release.yml
name: Release

on:
  push:
    branches: [main]

permissions:
  contents: write
  issues: write
  pull-requests: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # full history required for semantic-release

      - uses: actions/setup-node@v4
        with:
          node-version: 20

      - run: npm ci

      - run: npx semantic-release
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          NPM_TOKEN: ${{ secrets.NPM_TOKEN }}
```

## CHANGELOG.md Format (Keep a Changelog)

```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Nothing yet.

## [2.1.0] - 2024-01-15

### Added
- `GET /users?role=admin` filter parameter for listing users by role.
- `role` field on User object (`user` | `admin` | `moderator`).
- Rate limiting on authentication endpoints (10 requests/minute per IP).

### Changed
- `POST /users` now returns `201 Created` instead of `200 OK`.
- Improved error messages for validation failures — now includes field path.

### Fixed
- Fixed duplicate order creation when client retried on timeout.
- Fixed missing `Content-Type: application/json` header on error responses.

### Security
- Updated `jsonwebtoken` to 9.0.2 to address CVE-2022-23529.

## [2.0.0] - 2023-07-01

### Breaking Changes
- `GET /users` response now uses cursor-based pagination.
  - Before: `{ data: [], total: 100, page: 1, limit: 20 }`
  - After: `{ data: [], pageInfo: { hasNextPage, endCursor, totalCount } }`
- Removed `username` field from User object. Use `name` instead.

### Added
- GraphQL API at `/graphql` (Alpha).

### Deprecated
- REST v1 API (`/v1/`). Sunset date: 2024-12-31.

## [1.0.0] - 2023-01-01

### Added
- Initial release.
- User management: create, read, update, delete.
- JWT authentication.
- OpenAPI 3.x documentation at `/api-docs`.

[Unreleased]: https://github.com/myorg/myapp/compare/v2.1.0...HEAD
[2.1.0]: https://github.com/myorg/myapp/compare/v2.0.0...v2.1.0
[2.0.0]: https://github.com/myorg/myapp/compare/v1.0.0...v2.0.0
[1.0.0]: https://github.com/myorg/myapp/releases/tag/v1.0.0
```

## Key Rules

- Every PR must have at least one conventional commit — enforce with commitlint in CI.
- Use `feat!` or `BREAKING CHANGE:` footer for breaking changes — semantic-release bumps major version.
- Never manually edit CHANGELOG.md if using semantic-release — it will overwrite your edits.
- Add the `[skip ci]` flag to the semantic-release commit to prevent infinite CI loops.
- Scope names should be consistent across the team — document valid scopes in CONTRIBUTING.md.
- The `Unreleased` section in CHANGELOG.md should always exist and be kept up-to-date in PRs.
- Run `npx commitlint --from HEAD~1 --to HEAD` in CI to validate the latest commit on every push.
