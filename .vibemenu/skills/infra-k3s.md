# K3s Skill Guide

## Overview

K3s is a lightweight Kubernetes distribution by Rancher. It bundles Traefik ingress, local-path-provisioner for PVCs, and can run embedded etcd for HA. Ideal for edge, IoT, and small clusters.

## Installation

```bash
# Single-node server
curl -sfL https://get.k3s.io | sh -

# HA cluster — first server node (embedded etcd)
curl -sfL https://get.k3s.io | K3S_TOKEN=SECRET sh -s - server --cluster-init

# Additional server nodes
curl -sfL https://get.k3s.io | K3S_TOKEN=SECRET sh -s - server \
  --server https://FIRST_SERVER_IP:6443

# Agent (worker) node
curl -sfL https://get.k3s.io | K3S_URL=https://SERVER_IP:6443 K3S_TOKEN=SECRET sh -

# k3sup (easier installation tool)
k3sup install --ip SERVER_IP --user ubuntu
k3sup join --ip AGENT_IP --server-ip SERVER_IP --user ubuntu
```

## K3s-Specific Considerations

- **Traefik** is the default ingress controller (replaces nginx-ingress).
- **local-path-provisioner** handles PVCs with `storageClassName: local-path`.
- **SQLite** is the default datastore for single-node; use `--cluster-init` for embedded etcd HA.
- No cloud provider built in — no automatic LoadBalancer provisioning. Use MetalLB or Klipper LB.
- kubeconfig at `/etc/rancher/k3s/k3s.yaml`.

## Traefik IngressRoute

```yaml
# Native Traefik CRD (preferred over standard Ingress in K3s)
apiVersion: traefik.containo.us/v1alpha1
kind: IngressRoute
metadata:
  name: api-route
  namespace: default
spec:
  entryPoints:
    - web
    - websecure
  routes:
    - match: Host(`api.example.com`) && PathPrefix(`/`)
      kind: Rule
      services:
        - name: api
          port: 80
  tls:
    certResolver: letsencrypt

---
# Or use standard Ingress (Traefik also serves these)
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-ingress
  annotations:
    traefik.ingress.kubernetes.io/router.tls.certresolver: letsencrypt
spec:
  rules:
    - host: api.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: api
                port:
                  number: 80
```

## HelmChart CRD (K3s native app deployment)

```yaml
# K3s auto-deploys Helm charts defined as HelmChart CRDs
apiVersion: helm.cattle.io/v1
kind: HelmChart
metadata:
  name: my-app
  namespace: kube-system
spec:
  repo: https://charts.example.com
  chart: my-app
  targetNamespace: default
  valuesContent: |-
    replicaCount: 2
    image:
      tag: "1.2.3"
    service:
      type: ClusterIP
```

## PVC with local-path-provisioner

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: data-pvc
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: local-path   # K3s default
  resources:
    requests:
      storage: 5Gi
```

## Traefik TLS (Let's Encrypt via ACME)

Add to `/etc/rancher/k3s/config.yaml` or pass as flags:

```yaml
# /etc/rancher/k3s/config.yaml
write-kubeconfig-mode: "0644"
tls-san:
  - api.example.com
```

Configure Traefik ACME via HelmChartConfig:

```yaml
apiVersion: helm.cattle.io/v1
kind: HelmChartConfig
metadata:
  name: traefik
  namespace: kube-system
spec:
  valuesContent: |-
    additionalArguments:
      - "--certificatesresolvers.letsencrypt.acme.email=admin@example.com"
      - "--certificatesresolvers.letsencrypt.acme.storage=/data/acme.json"
      - "--certificatesresolvers.letsencrypt.acme.httpchallenge.entrypoint=web"
```

## Key Rules

- Use `local-path` storage class for single-node; for multi-node, deploy Longhorn or NFS provisioner.
- `--cluster-init` on first server creates embedded etcd; subsequent servers use `--server` flag to join.
- Traefik handles TLS termination natively — no cert-manager required unless you prefer it.
- K3s does not support cloud provider integrations — `LoadBalancer` services use Klipper (single-node) or MetalLB (multi-node).
- Store the K3s token (`/var/lib/rancher/k3s/server/node-token`) securely — it grants cluster join access.
