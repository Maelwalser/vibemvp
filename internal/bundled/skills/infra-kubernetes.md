# Kubernetes Skill Guide

## Overview

Kubernetes orchestrates containerized workloads. Key resource types: Deployment, Service, ConfigMap, Secret, Ingress, HPA, PVC, RBAC, NetworkPolicy.

## Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
  namespace: default
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: api
  template:
    metadata:
      labels:
        app: api
    spec:
      containers:
        - name: api
          image: ghcr.io/org/api:latest
          ports:
            - containerPort: 8080
          envFrom:
            - configMapRef:
                name: api-config
            - secretRef:
                name: api-secrets
          resources:
            requests:
              cpu: "100m"
              memory: "128Mi"
            limits:
              cpu: "500m"
              memory: "512Mi"
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /readyz
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
```

## Service Types

```yaml
# ClusterIP (internal only)
apiVersion: v1
kind: Service
metadata:
  name: api
spec:
  type: ClusterIP
  selector:
    app: api
  ports:
    - port: 80
      targetPort: 8080

# LoadBalancer (cloud provider external LB)
apiVersion: v1
kind: Service
metadata:
  name: api-lb
spec:
  type: LoadBalancer
  selector:
    app: api
  ports:
    - port: 80
      targetPort: 8080
```

## ConfigMap and Secret

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: api-config
data:
  APP_ENV: production
  LOG_LEVEL: info
  PORT: "8080"

---
apiVersion: v1
kind: Secret
metadata:
  name: api-secrets
type: Opaque
stringData:
  DATABASE_URL: postgres://user:pass@host:5432/db
  JWT_SECRET: changeme
```

## Ingress with TLS

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: api-ingress
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  ingressClassName: nginx
  tls:
    - hosts:
        - api.example.com
      secretName: api-tls
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

## HPA

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

## PVC

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: data-pvc
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: standard
  resources:
    requests:
      storage: 10Gi
```

## RBAC

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: api-sa
  namespace: default

---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: api-role
rules:
  - apiGroups: [""]
    resources: ["configmaps", "secrets"]
    verbs: ["get", "list"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: api-rolebinding
subjects:
  - kind: ServiceAccount
    name: api-sa
roleRef:
  kind: Role
  name: api-role
  apiGroup: rbac.authorization.k8s.io
```

## NetworkPolicy

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: api-netpol
spec:
  podSelector:
    matchLabels:
      app: api
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app: frontend
      ports:
        - port: 8080
  egress:
    - to:
        - podSelector:
            matchLabels:
              app: postgres
      ports:
        - port: 5432
```

## Key Rules

- Always set resource requests and limits — unbounded pods cause node pressure.
- Use `RollingUpdate` with `maxUnavailable: 0` for zero-downtime deployments.
- Store secrets as `Secret` objects, not `ConfigMap`. Use sealed-secrets or external-secrets-operator for GitOps.
- Add `readinessProbe` to prevent traffic before the pod is ready.
- Label everything consistently (`app`, `version`, `component`) for NetworkPolicy and HPA selectors.
- RBAC: grant minimal permissions per ServiceAccount; avoid `ClusterRole` for namespace-scoped needs.
