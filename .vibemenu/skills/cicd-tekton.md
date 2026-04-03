# Tekton CI/CD Skill Guide

## Overview

Tekton is a Kubernetes-native CI/CD framework. Build reusable Tasks, compose them into Pipelines, trigger via PipelineRuns or EventListeners.

## Task CRD

```yaml
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: go-test
spec:
  params:
    - name: package
      type: string
      description: Go package to test
      default: ./...
    - name: flags
      type: string
      default: "-v -race"

  workspaces:
    - name: source
      description: Source code workspace

  results:
    - name: test-output
      description: Test result output

  steps:
    - name: run-tests
      image: golang:1.22-alpine
      workingDir: $(workspaces.source.path)
      env:
        - name: GOFLAGS
          value: $(params.flags)
      script: |
        #!/bin/sh
        set -e
        go test $(params.package) 2>&1 | tee /tekton/results/test-output
```

```yaml
apiVersion: tekton.dev/v1
kind: Task
metadata:
  name: docker-build-push
spec:
  params:
    - name: image
      type: string
    - name: tag
      type: string
      default: latest

  workspaces:
    - name: source
    - name: dockerconfig
      description: Docker registry credentials

  steps:
    - name: build-push
      image: gcr.io/kaniko-project/executor:latest
      args:
        - --dockerfile=$(workspaces.source.path)/Dockerfile
        - --context=$(workspaces.source.path)
        - --destination=$(params.image):$(params.tag)
      volumeMounts:
        - name: kaniko-secret
          mountPath: /kaniko/.docker
  volumes:
    - name: kaniko-secret
      secret:
        secretName: regcred
        items:
          - key: .dockerconfigjson
            path: config.json
```

## Pipeline (composing Tasks)

```yaml
apiVersion: tekton.dev/v1
kind: Pipeline
metadata:
  name: ci-pipeline
spec:
  params:
    - name: repo-url
      type: string
    - name: revision
      type: string
    - name: image
      type: string

  workspaces:
    - name: shared-workspace
    - name: git-credentials

  tasks:
    - name: clone
      taskRef:
        name: git-clone
        kind: ClusterTask
      workspaces:
        - name: output
          workspace: shared-workspace
        - name: ssh-directory
          workspace: git-credentials
      params:
        - name: url
          value: $(params.repo-url)
        - name: revision
          value: $(params.revision)

    # Parallel execution (no runAfter = runs in parallel)
    - name: lint
      runAfter: [clone]
      taskRef:
        name: golangci-lint
      workspaces:
        - name: source
          workspace: shared-workspace

    - name: test
      runAfter: [clone]
      taskRef:
        name: go-test
      workspaces:
        - name: source
          workspace: shared-workspace

    # Sequenced after lint AND test both complete
    - name: build
      runAfter: [lint, test]
      taskRef:
        name: docker-build-push
      workspaces:
        - name: source
          workspace: shared-workspace
      params:
        - name: image
          value: $(params.image)
        - name: tag
          value: $(params.revision)

    # Conditional deploy
    - name: deploy
      runAfter: [build]
      when:
        - input: $(params.revision)
          operator: notin
          values: [""]
      taskRef:
        name: kubectl-deploy
      params:
        - name: image
          value: $(params.image):$(params.revision)
```

## PipelineRun

```yaml
apiVersion: tekton.dev/v1
kind: PipelineRun
metadata:
  name: ci-pipeline-run-001
spec:
  pipelineRef:
    name: ci-pipeline
  params:
    - name: repo-url
      value: https://github.com/org/api.git
    - name: revision
      value: abc1234
    - name: image
      value: ghcr.io/org/api
  workspaces:
    - name: shared-workspace
      persistentVolumeClaim:
        claimName: pipeline-pvc
    - name: git-credentials
      secret:
        secretName: github-ssh-key
```

## EventListener + Webhook Trigger

```yaml
apiVersion: triggers.tekton.dev/v1beta1
kind: TriggerTemplate
metadata:
  name: ci-trigger-template
spec:
  params:
    - name: git-revision
    - name: git-repo-url
  resourcetemplates:
    - apiVersion: tekton.dev/v1
      kind: PipelineRun
      metadata:
        generateName: ci-run-
      spec:
        pipelineRef:
          name: ci-pipeline
        params:
          - name: revision
            value: $(tt.params.git-revision)
          - name: repo-url
            value: $(tt.params.git-repo-url)

---
apiVersion: triggers.tekton.dev/v1beta1
kind: TriggerBinding
metadata:
  name: github-push-binding
spec:
  params:
    - name: git-revision
      value: $(body.head_commit.id)
    - name: git-repo-url
      value: $(body.repository.clone_url)

---
apiVersion: triggers.tekton.dev/v1beta1
kind: EventListener
metadata:
  name: github-listener
spec:
  triggers:
    - name: push-trigger
      bindings:
        - ref: github-push-binding
      template:
        ref: ci-trigger-template
      interceptors:
        - ref:
            name: github
          params:
            - name: secretRef
              value:
                secretName: github-webhook-secret
                secretKey: token
            - name: eventTypes
              value: ["push"]
```

## Key Rules

- Tasks are reusable building blocks — keep them single-purpose and parameterized.
- `runAfter` creates sequential dependencies; omitting it allows parallel execution.
- `when` conditions skip tasks based on param values — use for conditional deploy steps.
- Use `ClusterTask` for shared tasks across namespaces (e.g., git-clone from Tekton catalog).
- Workspaces share files between tasks in a pipeline — use PVC for large repos/artifacts.
- EventListener creates a Service that receives webhooks — expose via Ingress or port-forward for testing.
- Use Kaniko inside pipelines for Docker builds (no Docker-in-Docker socket mounting needed).
