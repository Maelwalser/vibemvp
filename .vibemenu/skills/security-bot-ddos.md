# Bot Management & DDoS Protection Skill Guide

## Overview

Layered defense against bot abuse and volumetric attacks: CDN/edge (Cloudflare, AWS Shield, GCP Cloud Armor) for L3/L4/L7 volumetric protection, behavioral ML (DataDome, Imperva) for sophisticated bots, and application-layer rate-based rules for abnormal traffic patterns.

---

## Cloudflare Bot Management

### Bot Score Thresholds

Cloudflare assigns each request a **bot score** (1 = definitely bot, 99 = definitely human).

```
# Expression language
# Score < 30 = likely bot
(cf.bot_management.score lt 30)

# Block verified bots that aren't allowlisted
(cf.bot_management.score lt 30 and not cf.bot_management.verified_bot)

# Challenge suspicious mid-range scores
(cf.bot_management.score ge 30 and cf.bot_management.score lt 60)
```

### Challenge Actions

```
# JS Challenge — runs JS in browser to confirm real browser
Action: js_challenge

# Managed Challenge — CF selects challenge type based on visitor history
Action: managed_challenge

# Block immediately
Action: block

# Log only (detection mode)
Action: log
```

### Recommended Bot Management Rules

```
Priority 1 — Block obvious bots:
  (cf.bot_management.score lt 10 and not cf.bot_management.verified_bot)
  → Action: block

Priority 2 — Challenge borderline requests:
  (cf.bot_management.score ge 10 and cf.bot_management.score lt 40
   and not cf.bot_management.verified_bot)
  → Action: managed_challenge

Priority 3 — Allow verified bots (Googlebot, etc.):
  (cf.bot_management.verified_bot)
  → Action: skip (bypass WAF)
```

### Under Attack Mode (L7 DDoS)

Enable via API when under active attack:
```bash
curl -X PATCH "https://api.cloudflare.com/client/v4/zones/{ZONE_ID}/settings/security_level" \
  -H "Authorization: Bearer $CF_TOKEN" \
  -H "Content-Type: application/json" \
  --data '{"value":"under_attack"}'
```

Security levels: `essentially_off` → `low` → `medium` → `high` → `under_attack`

Under Attack mode serves a JS interstitial to all visitors for 5 seconds before allowing access.

### L7 Managed Rules (HTTP DDoS)

```hcl
# Terraform
resource "cloudflare_ruleset" "ddos_l7" {
  zone_id = var.zone_id
  name    = "HTTP DDoS Attack Protection"
  kind    = "zone"
  phase   = "ddos_l7"

  rules {
    action = "execute"
    action_parameters {
      id = "4d21379b4f9f4bb088e0729962c8b3cf"  # CF HTTP DDoS ruleset
      overrides {
        sensitivity_level = "high"   # default / low / medium / high
        action            = "block"
      }
    }
    expression  = "true"
    description = "HTTP DDoS protection"
    enabled     = true
  }
}
```

---

## DataDome (Behavioral ML)

DataDome evaluates every request in <2ms using browser fingerprinting, behavioral signals, and ML models without requiring CAPTCHA for most users.

### Integration (NGINX module)

```nginx
# /etc/nginx/conf.d/datadome.conf
DataDomeServerSideKey "YOUR_SERVER_SIDE_KEY";
DataDomeUri "http://api.datadome.co/validate-request/";
DataDomeTimeoutMs 150;
DataDomeConnectTimeoutMs 100;
DataDomeFailOpen on;   # Fail open on timeout
```

### Node.js SDK

```typescript
import { datadomeMiddleware } from '@datadome/node'

app.use(datadomeMiddleware({
  apiKey: process.env.DATADOME_SERVER_KEY!,
  timeout: 150,
  failOpen: true,
}))
```

Responses: 200 (allow), 403 (block), 301/302 (redirect to CAPTCHA page).

---

## Imperva (Hi-Def Fingerprinting)

Imperva uses multi-layered fingerprinting: TLS fingerprint (JA3), HTTP/2 fingerprint, behavioral mouse/scroll patterns.

### Cloud WAF Integration

Route traffic through Imperva by updating DNS CNAME to Imperva PoP, then configure protection rules in the Imperva dashboard:

- Enable **Bot Access Control** policy
- Set **Threshold-Based Rules**: >500 req/min from single IP → challenge
- Enable **Advanced Bot Protection** for account takeover scenarios
- Configure **Allow List** for monitoring agents, health checks

---

## AWS Shield

### Shield Standard (Automatic, Always On)

Protects all AWS resources against **L3/L4** attacks (SYN floods, UDP reflection, volumetric) automatically at no additional cost. No configuration required.

### Shield Advanced (L7 ML Mitigation)

```hcl
resource "aws_shield_protection" "alb" {
  name         = "app-alb-shield"
  resource_arn = aws_lb.main.arn
}

resource "aws_shield_protection_group" "all_albs" {
  protection_group_id = "all-albs"
  aggregation         = "SUM"
  pattern             = "BY_RESOURCE_TYPE"
  resource_type       = "APPLICATION_LOAD_BALANCER"
}
```

Shield Advanced features:
- L7 ML-based detection for application-layer attacks
- DDoS cost protection (AWS credits anomalous charges during attacks)
- 24/7 DDoS Response Team (DRT) access
- Automatic WAF rule deployment during attacks

Pair with AWS WAF rate-based rule:
```hcl
rule {
  name     = "RateLimitPerIP"
  priority = 10
  action { block {} }

  statement {
    rate_based_statement {
      limit              = 2000  # requests per 5-minute window
      aggregate_key_type = "IP"
    }
  }
  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name                = "RateLimitPerIP"
    sampled_requests_enabled   = true
  }
}
```

---

## GCP Cloud Armor

```hcl
resource "google_compute_security_policy" "app_policy" {
  name = "app-security-policy"

  # Block known malicious IPs
  rule {
    action   = "deny(403)"
    priority = 1000
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["192.0.2.0/24"]
      }
    }
    description = "Block bad actor IP range"
  }

  # OWASP Top 10 preconfigured rules
  rule {
    action   = "deny(403)"
    priority = 2000
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('xss-v33-stable')"
      }
    }
  }

  rule {
    action   = "deny(403)"
    priority = 2001
    match {
      expr {
        expression = "evaluatePreconfiguredExpr('sqli-v33-stable')"
      }
    }
  }

  # Rate limiting rule (Cloud Armor Advanced)
  rule {
    action   = "rate_based_ban"
    priority = 3000
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    rate_limit_options {
      count          = 1000
      interval_sec   = 60
      ban_duration_sec = 300
      conform_action = "allow"
      exceed_action  = "deny(429)"
      enforce_on_key = "IP"
    }
  }

  # Default allow
  rule {
    action   = "allow"
    priority = 2147483647
    match {
      versioned_expr = "SRC_IPS_V1"
      config {
        src_ip_ranges = ["*"]
      }
    }
    description = "default allow"
  }
}

# Attach to backend service
resource "google_compute_backend_service" "app" {
  security_policy = google_compute_security_policy.app_policy.id
}
```

---

## Traffic Baseline & Anomaly Alerting

### Establish Baseline

```python
# Pseudocode: compute hourly p95 request rate over 7-day rolling window
baseline_p95 = percentile(hourly_request_counts_7d, 95)
alert_threshold = baseline_p95 * 2.0   # alert at 2x normal peak
```

### CloudWatch Metric Alarm (AWS)

```hcl
resource "aws_cloudwatch_metric_alarm" "ddos_anomaly" {
  alarm_name          = "ddos-traffic-spike"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = 2
  metric_name         = "RequestCount"
  namespace           = "AWS/ApplicationELB"
  period              = 60   # 1 minute
  statistic           = "Sum"
  threshold           = 10000  # Set from baseline analysis
  alarm_actions       = [aws_sns_topic.alerts.arn]

  dimensions = {
    LoadBalancer = aws_lb.main.arn_suffix
  }
}
```

### Cloudflare Analytics Alert

Configure via Cloudflare Notifications → Security Events:
- **Trigger:** HTTP DDoS Attack Alert → sensitivity High
- **Notification:** PagerDuty / Slack webhook
- **Alert on:** >10,000 blocked requests/minute (adjust to baseline)

---

## Rate-Based Rules (Application Layer)

Supplement CDN protection with application-layer rate-based blocking:

```nginx
# NGINX — limit_req for L7 rate limiting
limit_req_zone $binary_remote_addr zone=api:10m rate=100r/m;
limit_req_zone $http_x_api_key zone=apikey:10m rate=1000r/m;

server {
    location /api/ {
        limit_req zone=api burst=20 nodelay;
        limit_req_status 429;
    }
}
```

---

## Key Rules

- **Layer defense**: CDN-level (L3/L4) + WAF (L7) + application rate limiting — no single layer is sufficient
- **Fail open** on third-party bot detection service timeouts — availability > perfect enforcement
- **Allowlist** known-good bots (Googlebot, UptimeRobot, health checkers) before applying bot scoring
- **Establish baselines** before setting thresholds — alert at 2-3x p95, not arbitrary numbers
- **Under Attack mode** is a last resort — it adds 5s delay for all visitors including legitimate users
- Review bot management logs weekly; verified bot lists change as crawlers update their user agents
- Keep Shield Advanced enabled in regions serving production traffic — reactive enablement is too slow
- Test your DDoS response runbook quarterly with tabletop exercises
