# Mutual TLS (mTLS) Skill Guide

## Overview

Mutual TLS (mTLS) extends standard TLS by requiring the client to present a certificate in addition to the server. Both parties authenticate each other at the transport layer. This is the strongest form of transport-level authentication for service-to-service communication.

Use mTLS for internal microservice authentication, IoT device authentication, and high-security API access where shared secrets are undesirable.

---

## Implementation Pattern

### Certificate Provisioning

```bash
# 1. Create a Certificate Authority (CA)
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 3650 -key ca.key -out ca.crt \
  -subj "/O=Your Org/CN=Internal CA"

# 2. Generate a client key and CSR
openssl genrsa -out client.key 4096
openssl req -new -key client.key -out client.csr \
  -subj "/O=Your Org/CN=service-name"

# 3. CA signs the client certificate
openssl x509 -req -days 365 -in client.csr \
  -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out client.crt

# 4. Verify the cert
openssl verify -CAfile ca.crt client.crt
```

For production, use an automated PKI like CFSSL, Vault PKI, or cert-manager in Kubernetes.

### Go — TLS Server Requiring Client Certificate

```go
import (
    "crypto/tls"
    "crypto/x509"
    "os"
    "net/http"
)

func newMTLSServer(addr string, handler http.Handler) *http.Server {
    // Load CA cert pool for client verification
    caCert, err := os.ReadFile("ca.crt")
    if err != nil {
        panic(fmt.Errorf("load CA cert: %w", err))
    }
    caPool := x509.NewCertPool()
    if !caPool.AppendCertsFromPEM(caCert) {
        panic("failed to add CA cert to pool")
    }

    // Load server cert+key
    serverCert, err := tls.LoadX509KeyPair("server.crt", "server.key")
    if err != nil {
        panic(fmt.Errorf("load server cert: %w", err))
    }

    tlsCfg := &tls.Config{
        ClientAuth:   tls.RequireAndVerifyClientCert,
        ClientCAs:    caPool,
        Certificates: []tls.Certificate{serverCert},
        MinVersion:   tls.VersionTLS13,
    }

    return &http.Server{
        Addr:      addr,
        Handler:   handler,
        TLSConfig: tlsCfg,
    }
}
```

### Go — TLS Client with Client Certificate

```go
func newMTLSClient() *http.Client {
    clientCert, err := tls.LoadX509KeyPair("client.crt", "client.key")
    if err != nil {
        panic(fmt.Errorf("load client cert: %w", err))
    }

    caCert, _ := os.ReadFile("ca.crt")
    caPool := x509.NewCertPool()
    caPool.AppendCertsFromPEM(caCert)

    tlsCfg := &tls.Config{
        Certificates: []tls.Certificate{clientCert},
        RootCAs:      caPool,
        MinVersion:   tls.VersionTLS13,
    }
    return &http.Client{
        Transport: &http.Transport{TLSClientConfig: tlsCfg},
    }
}
```

### Extract Identity from Client Certificate

```go
// Middleware: extract CN from verified client cert
func clientIdentityMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
            http.Error(w, "client certificate required", http.StatusUnauthorized)
            return
        }
        cert := r.TLS.PeerCertificates[0]
        cn := cert.Subject.CommonName
        // Also inspect SANs: cert.DNSNames, cert.EmailAddresses, cert.URIs
        ctx := context.WithValue(r.Context(), "client_cn", cn)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

## nginx / Envoy / Traefik Configuration

### nginx mTLS

```nginx
server {
    listen 443 ssl;
    server_name api.internal;

    ssl_certificate     /etc/ssl/server.crt;
    ssl_certificate_key /etc/ssl/server.key;

    # Client cert verification
    ssl_client_certificate /etc/ssl/ca.crt;
    ssl_verify_client      on;
    ssl_verify_depth       2;

    ssl_protocols TLSv1.3;

    location / {
        # Forward verified client identity to upstream
        proxy_set_header X-Client-CN   $ssl_client_s_dn_cn;
        proxy_set_header X-Client-Cert $ssl_client_escaped_cert;
        proxy_pass http://upstream;
    }
}
```

### Envoy mTLS (downstream)

```yaml
transport_socket:
  name: envoy.transport_sockets.tls
  typed_config:
    "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext
    require_client_certificate: true
    common_tls_context:
      tls_certificates:
        - certificate_chain: { filename: /certs/server.crt }
          private_key: { filename: /certs/server.key }
      validation_context:
        trusted_ca: { filename: /certs/ca.crt }
```

### Kubernetes — cert-manager Client Certificate

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: service-client-cert
spec:
  secretName: service-client-tls
  duration: 720h      # 30 days
  renewBefore: 168h   # 7 days before expiry
  subject:
    organizations: [your-org]
  commonName: service-name
  isCA: false
  usages:
    - client auth
  issuerRef:
    name: internal-ca
    kind: ClusterIssuer
```

---

## Certificate Rotation Strategy

1. Issue new client certificate before the current one expires (renewBefore).
2. Deploy new cert alongside old cert without downtime (multi-cert support in client).
3. Server CA pool accepts both old and new certs during transition.
4. After all clients have rotated, remove old cert references.

```go
// Support multiple client certs during rotation
tlsCfg := &tls.Config{
    Certificates: []tls.Certificate{newCert, oldCert}, // try new first
    RootCAs:      caPool,
}
```

---

## Security Rules

- Never use self-signed certificates directly — always use a CA hierarchy.
- Pin the CA certificate; reject certificates from unknown CAs.
- Set `MinVersion: tls.VersionTLS13` — TLS 1.0 and 1.1 are deprecated.
- Rotate client certificates before expiry; automate with cert-manager or Vault.
- Validate the client CN/SAN against an allowlist — don't trust any CA-signed cert blindly.
- Never log or store the private key; use file permissions 600.
- Monitor certificate expiry and alert at least 30 days before expiration.

---

## Key Rules

- Both server and client present certificates — mutual verification at TLS handshake.
- CA signs all certificates; trust anchored to CA cert (`ClientCAs` / `RootCAs`).
- Extract identity from `cert.Subject.CommonName` or `cert.DNSNames` (SAN).
- TLS 1.3 minimum; reject older protocol versions.
- Automate rotation with cert-manager; set renewBefore to at least 25% of cert lifetime.
- nginx: `ssl_verify_client on`; Envoy: `require_client_certificate: true`.
