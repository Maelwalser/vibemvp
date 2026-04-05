# CI/CD Pipelines Skill Guide

## Overview

Automated pipelines enforce code quality before deployment. Standard stages: lint → typecheck → test → build → security-scan → deploy.

## GitHub Actions — Full Workflow

```yaml
name: CI/CD

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: npm
      - run: npm ci
      - run: npm run lint
      - run: npm run typecheck

  test:
    runs-on: ubuntu-latest
    needs: lint
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
          POSTGRES_DB: testdb
        ports:
          - 5432:5432
        options: >-
          --health-cmd "pg_isready -U test"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      redis:
        image: redis:7
        ports:
          - 6379:6379
        options: --health-cmd "redis-cli ping" --health-interval 5s

    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: 22
          cache: npm
      - run: npm ci
      - run: npm run test:coverage
        env:
          DATABASE_URL: postgres://test:test@localhost:5432/testdb
          REDIS_URL: redis://localhost:6379
      - uses: codecov/codecov-action@v4

  build:
    runs-on: ubuntu-latest
    needs: test
    permissions:
      contents: read
      packages: write
    steps:
      - uses: actions/checkout@v4
      - uses: docker/setup-buildx-action@v3
      - uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}
      - uses: docker/build-push-action@v5
        with:
          context: .
          push: ${{ github.ref == 'refs/heads/main' }}
          tags: |
            ghcr.io/${{ github.repository }}:latest
            ghcr.io/${{ github.repository }}:${{ github.sha }}
          cache-from: type=gha
          cache-to: type=gha,mode=max

  security-scan:
    runs-on: ubuntu-latest
    needs: build
    steps:
      - uses: actions/checkout@v4
      - uses: aquasecurity/trivy-action@master
        with:
          image-ref: ghcr.io/${{ github.repository }}:${{ github.sha }}
          format: sarif
          output: trivy-results.sarif
      - uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: trivy-results.sarif

  deploy:
    runs-on: ubuntu-latest
    needs: [build, security-scan]
    if: github.ref == 'refs/heads/main'
    environment: production
    steps:
      - uses: actions/checkout@v4
      - run: |
          # Deploy to your platform
          echo "Deploying ${{ github.sha }}"
```

## Matrix Strategy (multi-version)

```yaml
jobs:
  test:
    strategy:
      matrix:
        node-version: [18, 20, 22]
        os: [ubuntu-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: ${{ matrix.node-version }}
      - run: npm ci && npm test
```

## PR Preview Environments

```yaml
  preview:
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Deploy to Vercel preview
        uses: amondnet/vercel-action@v25
        with:
          vercel-token: ${{ secrets.VERCEL_TOKEN }}
          vercel-org-id: ${{ secrets.VERCEL_ORG_ID }}
          vercel-project-id: ${{ secrets.VERCEL_PROJECT_ID }}
      - name: Create Neon DB branch
        uses: neondatabase/create-branch-action@v5
        with:
          project_id: ${{ secrets.NEON_PROJECT_ID }}
          branch_name: pr-${{ github.event.number }}
          api_key: ${{ secrets.NEON_API_KEY }}
```

## Scheduled Workflows

```yaml
on:
  schedule:
    - cron: "0 6 * * 1"   # Every Monday at 6am UTC
  workflow_dispatch:        # Allow manual trigger

jobs:
  dependency-audit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: npm audit --audit-level=high
```

## GitLab CI (.gitlab-ci.yml)

```yaml
stages:
  - lint
  - test
  - build
  - deploy

variables:
  DOCKER_DRIVER: overlay2
  IMAGE: $CI_REGISTRY_IMAGE:$CI_COMMIT_SHA

lint:
  stage: lint
  image: node:22-alpine
  cache:
    paths: [node_modules/]
  script:
    - npm ci
    - npm run lint
    - npm run typecheck

test:
  stage: test
  image: node:22-alpine
  services:
    - postgres:16
  variables:
    POSTGRES_USER: test
    POSTGRES_PASSWORD: test
    DATABASE_URL: postgres://test:test@postgres/testdb
  script:
    - npm ci
    - npm run test:coverage
  coverage: '/Statements\s*:\s*(\d+\.\d+)%/'

build:
  stage: build
  image: docker:24
  services:
    - docker:24-dind
  script:
    - docker login -u $CI_REGISTRY_USER -p $CI_REGISTRY_PASSWORD $CI_REGISTRY
    - docker build -t $IMAGE .
    - docker push $IMAGE
  only:
    - main

deploy:
  stage: deploy
  script:
    - echo "Deploy $IMAGE"
  environment:
    name: production
  only:
    - main
```

## CircleCI (config.yml)

```yaml
version: 2.1

orbs:
  node: circleci/node@5
  docker: circleci/docker@2

jobs:
  test:
    docker:
      - image: cimg/node:22.0
      - image: cimg/postgres:16.0
        environment:
          POSTGRES_USER: test
          POSTGRES_PASSWORD: test
    steps:
      - checkout
      - node/install-packages:
          pkg-manager: npm
      - run: npm test

workflows:
  ci:
    jobs:
      - test
      - docker/publish:
          requires: [test]
          image: $DOCKER_LOGIN/$CIRCLE_PROJECT_REPONAME
          filters:
            branches:
              only: main
```

## Key Rules

- Gate the `deploy` job behind `build` and `security-scan` — never deploy unscanned images.
- Use `environment: production` in GitHub Actions to require approval for production deploys.
- Service containers in GitHub Actions need `health-cmd` options to avoid race conditions.
- Cache `node_modules` or Go module cache to speed up CI by 50-80%.
- Push Docker images only on `main` branch pushes, not on PRs.
- Use `cache-from: type=gha` with Docker BuildKit for layer caching across CI runs.
