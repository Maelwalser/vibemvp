# DevOps Standards Skill Guide

## Overview

Branch strategies (GitHub Flow, GitFlow, Trunk-based), Dependabot, Renovate, code review policies, and uptime/latency SLO recording rules.

## Branch Strategies

### GitHub Flow (recommended for most teams)

```
main (always deployable)
  └── feature/add-user-roles       ← short-lived, < 1 week
  └── fix/duplicate-order-bug
  └── chore/bump-dependencies
```

```yaml
# .github/branch-protection.yml (settings via GitHub API or Terraform)
# Branch protection for main:
required_status_checks:
  strict: true   # branch must be up-to-date before merge
  contexts:
    - ci/lint
    - ci/test
    - ci/build
    - security/scan

required_pull_request_reviews:
  required_approving_review_count: 1
  dismiss_stale_reviews: true          # re-review required after new push
  require_code_owner_reviews: true

restrictions: null  # allow all team members to push PRs
enforce_admins: true
allow_force_pushes: false
allow_deletions: false
```

### GitFlow (for versioned releases)

```
main         ← production releases tagged v1.0.0, v1.1.0
develop      ← integration branch, always ahead of main
  └── feature/TICKET-123-add-roles   ← branch from develop, merge to develop
  └── release/1.1.0                  ← branch from develop, merge to main+develop
  └── hotfix/1.0.1-fix-auth          ← branch from main, merge to main+develop
```

```bash
# GitFlow branch naming
git checkout -b feature/TICKET-123-add-roles develop
git checkout -b release/1.1.0 develop
git checkout -b hotfix/1.0.1-fix-auth main
```

### Trunk-Based Development (for high-frequency CI/CD)

```
main (trunk) ← everyone commits here, max 1-day branches
  └── dev/alice/fix-auth    ← max 24h lifespan
```

```typescript
// Feature flags replace long-lived branches
// new code ships hidden behind a flag
if (featureFlags.isEnabled('new-payment-flow', userId)) {
  return newPaymentFlow(req, res);
}
return legacyPaymentFlow(req, res);
```

## Dependabot

```yaml
# .github/dependabot.yml
version: 2
updates:
  # npm dependencies
  - package-ecosystem: npm
    directory: /
    schedule:
      interval: weekly
      day: monday
      time: "09:00"
      timezone: "America/New_York"
    groups:
      dev-dependencies:
        dependency-type: development
      production-minor:
        update-types: ["minor", "patch"]
    ignore:
      - dependency-name: "express"
        update-types: ["version-update:semver-major"]
    labels:
      - dependencies
      - automated
    open-pull-requests-limit: 10

  # Docker base images
  - package-ecosystem: docker
    directory: /
    schedule:
      interval: weekly
    labels:
      - dependencies
      - docker

  # Python
  - package-ecosystem: pip
    directory: /
    schedule:
      interval: weekly

  # GitHub Actions
  - package-ecosystem: github-actions
    directory: /
    schedule:
      interval: weekly
```

## Renovate

```json
// renovate.json
{
  "$schema": "https://docs.renovatebot.com/renovate-schema.json",
  "extends": [
    "config:recommended",
    ":dependencyDashboard",
    ":semanticCommits",
    ":separateMajorReleases",
    "group:monorepos"
  ],
  "schedule": ["before 9am on Monday"],
  "timezone": "America/New_York",
  "prConcurrentLimit": 5,
  "prHourlyLimit": 2,
  "labels": ["dependencies"],
  "packageRules": [
    {
      "description": "Auto-merge patch updates",
      "matchUpdateTypes": ["patch"],
      "automerge": true,
      "automergeType": "pr",
      "platformAutomerge": true
    },
    {
      "description": "Auto-merge minor devDependencies",
      "matchDepTypes": ["devDependencies"],
      "matchUpdateTypes": ["minor"],
      "automerge": true
    },
    {
      "description": "Group all AWS SDK updates",
      "matchPackagePrefixes": ["@aws-sdk/"],
      "groupName": "AWS SDK"
    },
    {
      "description": "Group all testing tools",
      "matchPackageNames": ["vitest", "jest", "playwright", "cypress", "@testing-library/*"],
      "groupName": "Testing tools"
    },
    {
      "description": "Hold major updates for manual review",
      "matchUpdateTypes": ["major"],
      "automerge": false,
      "addLabels": ["major-upgrade"]
    }
  ]
}
```

## Code Review Policy

```yaml
# CODEOWNERS — auto-assigns reviewers
# .github/CODEOWNERS
*                    @myorg/backend-team       # default reviewer for all files
/src/frontend/**     @myorg/frontend-team
/src/auth/**         @myorg/security-team
/terraform/**        @myorg/platform-team
/docs/**             @myorg/docs-team
```

```markdown
## Code Review Standards

### Required Approvals
- Low risk (docs, tests, chore): 1 approval
- Medium risk (new feature, refactor): 1 approval from CODEOWNER
- High risk (auth, payments, migrations): 2 approvals including security-team

### Checklist for reviewers
- [ ] Tests cover the change (unit + integration)
- [ ] No sensitive data in logs/responses
- [ ] Backward compatible OR migration plan documented
- [ ] Performance implications considered
- [ ] Error handling is complete
- [ ] No hardcoded secrets or credentials

### Rules
- Conversation resolution required before merge
- Dismiss stale reviews when new commits are pushed
- Never self-approve
- Squash-merge feature branches to keep main history clean
```

## Uptime SLO Recording Rules (Prometheus)

```yaml
# prometheus/rules/slo-devops.yml
groups:
  - name: uptime-slo
    rules:
      # 5-minute availability window
      - record: job:uptime:ratio_rate5m
        expr: |
          avg by (job, env) (up{job=~"myapp.*"})

      # 30-day rolling availability
      - record: job:uptime:ratio_rate30d
        expr: |
          avg_over_time(up{job=~"myapp.*"}[30d])

      # SLO compliance check
      - alert: UptimeSLOBreach99_9
        expr: job:uptime:ratio_rate30d < 0.999
        for: 5m
        labels:
          severity: critical
          slo: uptime
        annotations:
          summary: "Uptime below 99.9% SLO for {{ $labels.job }}"
          description: "30-day uptime is {{ $value | humanizePercentage }}"

      - alert: UptimeSLOBreach99_95
        expr: job:uptime:ratio_rate30d < 0.9995
        for: 5m
        labels:
          severity: warning
          slo: uptime
        annotations:
          summary: "Uptime below 99.95% SLO for {{ $labels.job }}"
```

## Latency P99 Alert (histogram_quantile)

```yaml
groups:
  - name: latency-slo
    rules:
      # P99 latency recording rule
      - record: job:request_latency_p99:rate5m
        expr: |
          histogram_quantile(0.99,
            sum by (job, env, le) (
              rate(http_request_duration_seconds_bucket[5m])
            )
          )

      # Alert: P99 > 500ms
      - alert: HighLatencyP99
        expr: job:request_latency_p99:rate5m > 0.5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "P99 latency above 500ms for {{ $labels.job }}"
          description: "P99 is {{ $value | humanizeDuration }}"

      # Alert: P99 > 2s (critical)
      - alert: CriticalLatencyP99
        expr: job:request_latency_p99:rate5m > 2.0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "P99 latency above 2s for {{ $labels.job }}"
```

## Key Rules

- **GitHub Flow**: require status checks + dismiss-stale-reviews + CODEOWNERS on `main`.
- **GitFlow**: never commit directly to `main` or `develop` — always use branch + PR.
- **Trunk-based**: enforce < 24h branch lifetime; use feature flags for incomplete features.
- **Dependabot / Renovate**: auto-merge patch updates; require review for minor and major.
- **Code review**: require at minimum 1 CODEOWNER approval; 2 for security-sensitive paths.
- Always squash-merge feature branches — preserves a clean, meaningful commit history on main.
- Run `buf breaking`, `openapi-validator`, and migration diff checks as required status checks.
- Track P99 latency and uptime in Grafana dashboards visible to the whole engineering team.
