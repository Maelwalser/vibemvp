# WAF (Web Application Firewall) Skill Guide

## Overview

Web Application Firewalls protect against OWASP Top 10 attacks, injection, XSS, and volumetric abuse. Three primary deployment surfaces: Cloudflare WAF (SaaS edge), AWS WAF (cloud-native), and ModSecurity (self-hosted). Rules evaluate at L7 before traffic reaches origin.

## Cloudflare WAF

### Custom Rule Expression Language

```
# Block requests to admin paths from non-allowlisted IPs
(http.request.uri.path contains "/admin" and not ip.src in {203.0.113.10 203.0.113.11})

# Block suspicious user-agents and methods
(http.user_agent contains "sqlmap" or http.user_agent contains "nikto")

# Country-based block with path exception
(ip.geoip.country in {"CN" "RU"} and not http.request.uri.path eq "/api/public")

# Rate limit payload: block large POST bodies
(http.request.method eq "POST" and http.request.body.size gt 1048576)
```

Actions: `block`, `challenge`, `js_challenge`, `managed_challenge`, `log`, `skip`.

### OWASP Managed Ruleset

Enable via Cloudflare dashboard → Security → WAF → Managed Rules → OWASP Core Ruleset.

Paranoia level controls rule sensitivity:
- **PL1** — Low false positives, catches obvious attacks only
- **PL2** — Recommended for most applications
- **PL3** — Stricter; may require exclusions for complex apps
- **PL4** — Maximum sensitivity; expect false positives

Set score threshold (default 25 — lower = more blocking):
```
# Terraform (Cloudflare provider)
resource "cloudflare_ruleset" "owasp" {
  zone_id = var.zone_id
  name    = "OWASP Managed Ruleset"
  kind    = "zone"
  phase   = "http_request_firewall_managed"

  rules {
    action = "execute"
    action_parameters {
      id = "4814384a9e5d4991b9815dcfc25d2f1f"  # OWASP ruleset ID
      overrides {
        sensitivity_level = "medium"   # low / medium / high
        action            = "block"
      }
    }
    expression  = "true"
    description = "OWASP Core Ruleset"
    enabled     = true
  }
}
```

### Rule Exclusion for False Positives

```
# Skip WAF managed rules for specific path (e.g., legacy API accepting raw JSON)
(http.request.uri.path eq "/api/v1/legacy" and http.request.method eq "POST")
→ Action: Skip → Managed rules: Skip all managed rules
```

Via Terraform:
```hcl
rules {
  action = "skip"
  action_parameters {
    ruleset = "current"
  }
  expression  = "(http.request.uri.path eq \"/api/upload\" and http.request.method eq \"POST\")"
  description = "Allow large uploads to bypass WAF body inspection"
  enabled     = true
}
```

### Audit Log Monitoring

Enable Logpush to S3/R2/Splunk for WAF events:
```json
{
  "dataset": "firewall_events",
  "fields": ["Action", "ClientIP", "ClientRequestPath", "RuleID", "RuleMessage", "Datetime"]
}
```

Alert on spikes: if WAF blocks exceed 2x 24h baseline → PagerDuty/Slack alert.

---

## AWS WAF

### WebACL Structure

```hcl
resource "aws_wafv2_web_acl" "main" {
  name  = "app-waf"
  scope = "REGIONAL"  # or CLOUDFRONT for distributions

  default_action {
    allow {}
  }

  # AWS Managed Rule Groups
  rule {
    name     = "AWSManagedRulesCommonRuleSet"
    priority = 1
    override_action { none {} }

    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesCommonRuleSet"
        vendor_name = "AWS"

        # Exclude rules causing false positives
        excluded_rule {
          name = "SizeRestrictions_BODY"
        }
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "CommonRuleSet"
      sampled_requests_enabled   = true
    }
  }

  rule {
    name     = "AWSManagedRulesSQLiRuleSet"
    priority = 2
    override_action { none {} }
    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesSQLiRuleSet"
        vendor_name = "AWS"
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "SQLiRuleSet"
      sampled_requests_enabled   = true
    }
  }

  # IP Set block rule
  rule {
    name     = "BlockBadIPs"
    priority = 0
    action { block {} }
    statement {
      ip_set_reference_statement {
        arn = aws_wafv2_ip_set.blocklist.arn
      }
    }
    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "BlockBadIPs"
      sampled_requests_enabled   = true
    }
  }

  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name                = "AppWAF"
    sampled_requests_enabled   = true
  }
}
```

### IP Set Management

```hcl
resource "aws_wafv2_ip_set" "blocklist" {
  name               = "bad-actor-ips"
  scope              = "REGIONAL"
  ip_address_version = "IPV4"
  addresses          = ["192.0.2.0/24", "198.51.100.44/32"]
}
```

### Associate with ALB or CloudFront

```hcl
# ALB association
resource "aws_wafv2_web_acl_association" "alb" {
  resource_arn = aws_lb.main.arn
  web_acl_arn  = aws_wafv2_web_acl.main.arn
}

# CloudFront: set web_acl_id directly on distribution
resource "aws_cloudfront_distribution" "main" {
  web_acl_id = aws_wafv2_web_acl.main.arn
  # scope must be CLOUDFRONT, deployed in us-east-1
}
```

---

## ModSecurity (NGINX/Apache)

### CRS Paranoia Levels

Install ModSecurity + OWASP CRS, then configure in `modsecurity.conf` / `crs-setup.conf`:

```apache
# crs-setup.conf
SecAction \
  "id:900000, \
  phase:1, \
  pass, \
  t:none, \
  nolog, \
  setvar:tx.paranoia_level=2"

# PL1 = essential rules only (fewest false positives)
# PL2 = recommended default
# PL3 = strict; enables additional detection
# PL4 = maximum; very high false positive risk
```

### NGINX ModSecurity Config

```nginx
modsecurity on;
modsecurity_rules_file /etc/nginx/modsec/modsecurity.conf;

# Audit log
SecAuditLog /var/log/modsec_audit.log
SecAuditLogParts ABIJDEFHZ
SecAuditEngine RelevantOnly
SecAuditLogRelevantStatus "^(?:5|4(?!04))"
```

### Rule Exclusion for False Positives

```apache
# Exclude rule 942100 (SQL injection) for /api/search path
SecRuleUpdateTargetById 942100 "!REQUEST_URI:/api/search"

# Disable rule entirely for a route
SecRule REQUEST_URI "@beginsWith /api/legacy" \
  "id:1001, phase:1, pass, nolog, \
  ctl:ruleRemoveById=942100"
```

---

## Key Rules

- Start with **PL1 or PL2** and increase after tuning — do not enable PL4 without thorough testing
- Always run WAF in **detection/log mode** first, then switch to block after baseline
- Log every blocked request with IP, rule ID, path, and timestamp for audit
- Review false positives weekly in the first month after deployment
- Rotate IP blocklists regularly; stale blocks waste rule evaluation cycles
- Never expose WAF configuration or rule IDs in error responses to clients
- Use separate WebACLs for internal APIs vs public-facing endpoints (different risk profiles)
