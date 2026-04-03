# Networking & Reverse Proxy Skill Guide

## Overview

Covers: Nginx/Caddy reverse proxy, Cloudflare Tunnel, Route53 DNS, Let's Encrypt via cert-manager, CloudFront CDN, and SSL auto-renewal.

## Nginx Reverse Proxy

```nginx
# /etc/nginx/conf.d/api.conf
upstream api_backend {
    least_conn;
    server api-1:8080 max_fails=3 fail_timeout=30s;
    server api-2:8080 max_fails=3 fail_timeout=30s;
    keepalive 32;
}

server {
    listen 80;
    server_name api.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.example.com;

    ssl_certificate     /etc/letsencrypt/live/api.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.example.com/privkey.pem;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;

    location / {
        proxy_pass http://api_backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Connection "";   # keep-alive to upstream
        proxy_read_timeout 60s;
        proxy_connect_timeout 5s;
    }

    location /healthz {
        proxy_pass http://api_backend;
        access_log off;
    }
}
```

## Caddy (automatic HTTPS)

```
# Caddyfile
api.example.com {
    reverse_proxy api-1:8080 api-2:8080 {
        lb_policy least_conn
        health_uri /healthz
        health_interval 10s
        transport http {
            keepalive 30s
        }
    }

    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
        X-Content-Type-Options nosniff
        X-Frame-Options DENY
    }

    log {
        output file /var/log/caddy/access.log {
            roll_size 10mb
            roll_keep 5
        }
        format json
    }
}
```

## Cloudflare Tunnel (no port forwarding)

```bash
# Install cloudflared
curl -L --output cloudflared.deb \
  https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
dpkg -i cloudflared.deb

# Authenticate and create tunnel
cloudflared tunnel login
cloudflared tunnel create my-tunnel
# Note the tunnel UUID

# Configure ingress rules
cat > ~/.cloudflared/config.yml <<EOF
tunnel: <TUNNEL-UUID>
credentials-file: /root/.cloudflared/<TUNNEL-UUID>.json

ingress:
  - hostname: api.example.com
    service: http://localhost:8080
  - hostname: admin.example.com
    service: http://localhost:3000
  - service: http_status:404
EOF

# Add DNS record (no IP needed!)
cloudflared tunnel route dns my-tunnel api.example.com

# Run as service
cloudflared service install
systemctl enable --now cloudflared
```

## Route53 DNS Records

```hcl
# Terraform — Route53 records
resource "aws_route53_record" "api_a" {
  zone_id = aws_route53_zone.main.zone_id
  name    = "api.example.com"
  type    = "A"
  alias {
    name                   = aws_lb.api.dns_name
    zone_id                = aws_lb.api.zone_id
    evaluate_target_health = true
  }
}

# Weighted routing (for blue/green)
resource "aws_route53_record" "api_blue" {
  zone_id        = aws_route53_zone.main.zone_id
  name           = "api.example.com"
  type           = "A"
  set_identifier = "blue"
  weighted_routing_policy {
    weight = 100
  }
  alias {
    name                   = aws_lb.blue.dns_name
    zone_id                = aws_lb.blue.zone_id
    evaluate_target_health = true
  }
}

# Latency-based routing
resource "aws_route53_record" "api_us" {
  zone_id        = aws_route53_zone.main.zone_id
  name           = "api.example.com"
  type           = "A"
  set_identifier = "us-east-1"
  latency_routing_policy {
    region = "us-east-1"
  }
  alias {
    name                   = aws_lb.us_east.dns_name
    zone_id                = aws_lb.us_east.zone_id
    evaluate_target_health = true
  }
}
```

## cert-manager (Let's Encrypt in Kubernetes)

```yaml
# ClusterIssuer — HTTP01 challenge
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
      - http01:
          ingress:
            class: nginx

---
# ClusterIssuer — DNS01 challenge (wildcard certs)
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-dns
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: admin@example.com
    privateKeySecretRef:
      name: letsencrypt-dns
    solvers:
      - dns01:
          route53:
            region: us-east-1
            hostedZoneID: XXXXXXXX

---
# Certificate resource
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: api-cert
spec:
  secretName: api-tls
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
    - api.example.com
    - "*.api.example.com"
```

## CloudFront Distribution

```hcl
# Terraform — CloudFront
resource "aws_cloudfront_distribution" "api" {
  origin {
    domain_name = aws_lb.api.dns_name
    origin_id   = "api-alb"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  enabled             = true
  is_ipv6_enabled     = true
  default_root_object = ""

  default_cache_behavior {
    allowed_methods        = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods         = ["GET", "HEAD"]
    target_origin_id       = "api-alb"
    viewer_protocol_policy = "redirect-to-https"
    compress               = true

    cache_policy_id          = "4135ea2d-6df8-44a3-9df3-4b5a84be39ad"  # CachingDisabled for API
    origin_request_policy_id = "b689b0a8-53d0-40ab-baf2-68738e2966ac"  # AllViewer

    # WAF
    # web_acl_id = aws_wafv2_web_acl.main.arn
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate.api.arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }
}
```

## SSL Auto-Renewal with Certbot

```bash
# Install certbot
apt install certbot python3-certbot-nginx

# Issue cert (Nginx plugin auto-configures)
certbot --nginx -d api.example.com -d www.example.com \
  --email admin@example.com --agree-tos --no-eff-email

# Auto-renewal (added by certbot automatically)
# /etc/cron.d/certbot: 0 */12 * * * root certbot renew --quiet

# Test renewal
certbot renew --dry-run
```

## Key Rules

- Always redirect HTTP to HTTPS — never serve plaintext in production.
- Set `X-Forwarded-For` and `X-Real-IP` headers so apps see the real client IP behind the proxy.
- Cloudflare Tunnel requires no open inbound ports — ideal for services behind NAT.
- cert-manager HTTP01 challenge requires port 80 to be reachable from the internet; use DNS01 for private clusters.
- Route53 alias records (not CNAME) are required for zone apex (example.com) pointing to ALBs.
- CloudFront: use `CachingDisabled` policy for API endpoints that must not be cached.
- Set TLS minimum version to TLSv1.2 — TLS 1.0/1.1 are deprecated and insecure.
