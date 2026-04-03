# GitOps with ArgoCD Skill Guide

## Overview

GitOps uses Git as the single source of truth for infrastructure. ArgoCD continuously syncs Kubernetes cluster state to match the Git repository.

## ArgoCD Application CRD

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: api-prod
  namespace: argocd
spec:
  project: team-backend

  source:
    repoURL: https://github.com/org/infra
    targetRevision: main
    path: apps/api/overlays/prod

  destination:
    server: https://kubernetes.default.svc
    namespace: api-prod

  syncPolicy:
    automated:
      prune: true        # delete resources removed from Git
      selfHeal: true     # revert manual kubectl changes
    syncOptions:
      - CreateNamespace=true
      - PrunePropagationPolicy=foreground
      - ApplyOutOfSyncOnly=true
    retry:
      limit: 5
      backoff:
        duration: 5s
        factor: 2
        maxDuration: 3m

  ignoreDifferences:
    - group: apps
      kind: Deployment
      jsonPointers:
        - /spec/replicas   # allow HPA to manage replicas
```

## AppProject (Team Access Control)

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: team-backend
  namespace: argocd
spec:
  description: Backend team project

  sourceRepos:
    - https://github.com/org/infra

  destinations:
    - namespace: api-*
      server: https://kubernetes.default.svc

  clusterResourceWhitelist:
    - group: ""
      kind: Namespace

  namespaceResourceWhitelist:
    - group: apps
      kind: Deployment
    - group: ""
      kind: Service
    - group: networking.k8s.io
      kind: Ingress

  roles:
    - name: deploy
      description: Can sync apps
      policies:
        - p, proj:team-backend:deploy, applications, sync, team-backend/*, allow
      groups:
        - org:backend-team
```

## Multi-Environment with Kustomize

```
apps/api/
├── base/
│   ├── kustomization.yaml
│   ├── deployment.yaml
│   ├── service.yaml
│   └── configmap.yaml
└── overlays/
    ├── dev/
    │   ├── kustomization.yaml
    │   └── patch-replicas.yaml
    ├── staging/
    │   ├── kustomization.yaml
    │   └── patch-resources.yaml
    └── prod/
        ├── kustomization.yaml
        ├── patch-replicas.yaml
        └── patch-resources.yaml
```

```yaml
# base/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - deployment.yaml
  - service.yaml
  - configmap.yaml
commonLabels:
  app: api

---
# overlays/prod/kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: api-prod
namePrefix: prod-
bases:
  - ../../base
patches:
  - path: patch-replicas.yaml
  - path: patch-resources.yaml
images:
  - name: ghcr.io/org/api
    newTag: "1.2.3"

---
# overlays/prod/patch-replicas.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 5
```

## ArgoCD Image Updater (automated version bumping)

```yaml
# Annotation on ArgoCD Application
metadata:
  annotations:
    argocd-image-updater.argoproj.io/image-list: api=ghcr.io/org/api
    argocd-image-updater.argoproj.io/api.update-strategy: semver
    argocd-image-updater.argoproj.io/api.allow-tags: regexp:^v\d+\.\d+\.\d+$
    argocd-image-updater.argoproj.io/write-back-method: git
    argocd-image-updater.argoproj.io/git-branch: main
```

## ArgoCD Notifications (Slack on sync fail)

```yaml
# argocd-notifications-cm ConfigMap
apiVersion: v1
kind: ConfigMap
metadata:
  name: argocd-notifications-cm
  namespace: argocd
data:
  trigger.on-sync-failed: |
    - when: app.status.operationState.phase in ['Error', 'Failed']
      send: [slack-sync-failed]

  template.slack-sync-failed: |
    message: |
      Application {{.app.metadata.name}} sync failed.
      Revision: {{.app.status.operationState.syncResult.revision}}

  service.slack: |
    token: $slack-token
    channels:
      - alerts-deployments
```

## ArgoCD CLI Essentials

```bash
# Install
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# Login
argocd login argocd.example.com --username admin --password $(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d)

# Sync an application manually
argocd app sync api-prod

# Check sync status
argocd app get api-prod

# Rollback to previous revision
argocd app rollback api-prod 1
```

## Key Rules

- `prune: true` + `selfHeal: true` = full GitOps — Git is authoritative, cluster state is always corrected.
- Add `ignoreDifferences` for HPA-managed fields (replicas) so ArgoCD doesn't fight the autoscaler.
- Use AppProject to restrict which repos, namespaces, and resource types each team can manage.
- Kustomize overlays allow environment-specific config without duplicating base manifests.
- Image Updater writes back to Git — every deployment is tracked in version history.
- Never `kubectl apply` directly in GitOps mode — all changes must go through Git PRs.
