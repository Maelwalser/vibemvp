# Cloud Run Skill Guide

## Overview

Cloud Run runs stateless containers on GCP with automatic scaling to zero. Supports HTTP, gRPC, and WebSocket. Configure concurrency, timeouts, CPU/memory, and traffic splitting.

## Service YAML

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: api
  namespace: default
  annotations:
    run.googleapis.com/ingress: all          # or: internal, internal-and-cloud-load-balancing
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/minScale: "1"
        autoscaling.knative.dev/maxScale: "100"
        run.googleapis.com/vpc-access-connector: projects/PROJECT/locations/REGION/connectors/my-connector
        run.googleapis.com/vpc-access-egress: all-traffic
    spec:
      serviceAccountName: api-sa@PROJECT.iam.gserviceaccount.com
      containerConcurrency: 80
      timeoutSeconds: 60
      containers:
        - image: REGION-docker.pkg.dev/PROJECT/repo/api:latest
          resources:
            limits:
              cpu: "2"
              memory: 512Mi
          env:
            - name: APP_ENV
              value: production
            - name: LOG_LEVEL
              value: info
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: db-url
                  key: latest
          ports:
            - containerPort: 8080
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8080
            initialDelaySeconds: 10
          startupProbe:
            httpGet:
              path: /healthz
              port: 8080
            failureThreshold: 3
            periodSeconds: 10
```

## Deploy via gcloud

```bash
gcloud run deploy api \
  --image REGION-docker.pkg.dev/PROJECT/repo/api:latest \
  --region us-central1 \
  --platform managed \
  --service-account api-sa@PROJECT.iam.gserviceaccount.com \
  --set-secrets="DATABASE_URL=db-url:latest" \
  --set-env-vars="APP_ENV=production" \
  --min-instances=1 \
  --max-instances=100 \
  --concurrency=80 \
  --timeout=60 \
  --allow-unauthenticated

# Deploy new revision and split traffic (canary)
gcloud run deploy api --image ...tag-v2...
gcloud run services update-traffic api \
  --to-revisions=api-v1=90,api-v2=10 \
  --region us-central1

# Promote canary to 100%
gcloud run services update-traffic api \
  --to-latest \
  --region us-central1
```

## VPC Connector (private networking)

```bash
# Create VPC connector for private CloudSQL / Memorystore access
gcloud compute networks vpc-access connectors create my-connector \
  --region us-central1 \
  --subnet my-subnet \
  --min-instances 2 \
  --max-instances 10
```

## Service Account with Minimal Permissions

```bash
# Create dedicated service account
gcloud iam service-accounts create api-sa \
  --display-name "Cloud Run API Service Account"

# Grant only what is needed
gcloud projects add-iam-policy-binding PROJECT \
  --member="serviceAccount:api-sa@PROJECT.iam.gserviceaccount.com" \
  --role="roles/cloudsql.client"

gcloud secrets add-iam-policy-binding db-url \
  --member="serviceAccount:api-sa@PROJECT.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

## Terraform — google_cloud_run_service

```hcl
resource "google_cloud_run_service" "api" {
  name     = "api"
  location = "us-central1"

  template {
    metadata {
      annotations = {
        "autoscaling.knative.dev/minScale" = "1"
        "autoscaling.knative.dev/maxScale" = "50"
        "run.googleapis.com/vpc-access-connector" = google_vpc_access_connector.connector.id
        "run.googleapis.com/vpc-access-egress"    = "all-traffic"
      }
    }
    spec {
      service_account_name  = google_service_account.api_sa.email
      container_concurrency = 80
      timeout_seconds       = 60
      containers {
        image = "gcr.io/PROJECT/api:latest"
        resources {
          limits = {
            cpu    = "2"
            memory = "512Mi"
          }
        }
        env {
          name  = "APP_ENV"
          value = "production"
        }
        env {
          name = "DATABASE_URL"
          value_from {
            secret_key_ref {
              name = google_secret_manager_secret.db_url.secret_id
              key  = "latest"
            }
          }
        }
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }
}

resource "google_cloud_run_service_iam_member" "public" {
  service  = google_cloud_run_service.api.name
  location = google_cloud_run_service.api.location
  role     = "roles/run.invoker"
  member   = "allUsers"   # remove for private services
}
```

## Key Rules

- Set `containerConcurrency` based on how many simultaneous requests one container handles safely (default 80).
- Use `--min-instances=1` to avoid cold starts for user-facing services.
- Always use Secret Manager via `secretKeyRef` — never put secrets in env var values directly.
- VPC connector is required for private CloudSQL, Memorystore, or internal services.
- Canary deployments: deploy new revision, split traffic by percentage, monitor, then promote.
- CPU is only allocated during request processing by default; set `run.googleapis.com/cpu-throttling: "false"` for background work.
