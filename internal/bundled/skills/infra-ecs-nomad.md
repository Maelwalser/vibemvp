# ECS & Nomad Skill Guide

## Overview

ECS (Elastic Container Service) runs containers on AWS. Nomad is a HashiCorp workload orchestrator supporting Docker, Java, and other drivers. Both handle scheduling, health checks, and service discovery.

## ECS Task Definition

```json
{
  "family": "api",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "executionRoleArn": "arn:aws:iam::123456789:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::123456789:role/ecsTaskRole",
  "containerDefinitions": [
    {
      "name": "api",
      "image": "123456789.dkr.ecr.us-east-1.amazonaws.com/api:latest",
      "essential": true,
      "portMappings": [
        { "containerPort": 8080, "protocol": "tcp" }
      ],
      "environment": [
        { "name": "APP_ENV", "value": "production" },
        { "name": "LOG_LEVEL", "value": "info" }
      ],
      "secrets": [
        {
          "name": "DATABASE_URL",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:123456789:secret:prod/api/db"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/api",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "api"
        }
      },
      "healthCheck": {
        "command": ["CMD-SHELL", "curl -f http://localhost:8080/healthz || exit 1"],
        "interval": 30,
        "timeout": 5,
        "retries": 3,
        "startPeriod": 60
      }
    }
  ]
}
```

## ECS Service with ALB and Auto-Scaling

```json
{
  "serviceName": "api",
  "cluster": "prod-cluster",
  "taskDefinition": "api:42",
  "launchType": "FARGATE",
  "desiredCount": 2,
  "networkConfiguration": {
    "awsvpcConfiguration": {
      "subnets": ["subnet-aaa", "subnet-bbb"],
      "securityGroups": ["sg-api"],
      "assignPublicIp": "DISABLED"
    }
  },
  "loadBalancers": [
    {
      "targetGroupArn": "arn:aws:elasticloadbalancing:...:targetgroup/api/xxx",
      "containerName": "api",
      "containerPort": 8080
    }
  ],
  "deploymentConfiguration": {
    "minimumHealthyPercent": 100,
    "maximumPercent": 200
  }
}
```

## ECS IAM Roles

```
Task Execution Role (ecsTaskExecutionRole):
  - ecr:GetAuthorizationToken
  - ecr:BatchGetImage / ecr:GetDownloadUrlForLayer
  - secretsmanager:GetSecretValue
  - logs:CreateLogStream / logs:PutLogEvents

Task Role (ecsTaskRole — app permissions):
  - s3:GetObject / s3:PutObject (if app reads/writes S3)
  - sqs:SendMessage / sqs:ReceiveMessage (if app uses SQS)
  - dynamodb:GetItem / dynamodb:PutItem (if app uses DynamoDB)
```

## Fargate vs EC2 Launch Types

| Aspect | Fargate | EC2 |
|--------|---------|-----|
| Server management | None (serverless) | Manage EC2 instances |
| Cost | Per vCPU+memory-second | Instance reservation |
| Use case | Stateless APIs, microservices | GPU workloads, large/long jobs |
| Networking | awsvpc only | bridge/host/awsvpc |

## ECS Auto-Scaling

```bash
# Register scalable target
aws application-autoscaling register-scalable-target \
  --service-namespace ecs \
  --scalable-dimension ecs:service:DesiredCount \
  --resource-id service/prod-cluster/api \
  --min-capacity 2 --max-capacity 10

# CPU utilization policy
aws application-autoscaling put-scaling-policy \
  --policy-name api-cpu-scaling \
  --service-namespace ecs \
  --resource-id service/prod-cluster/api \
  --scalable-dimension ecs:service:DesiredCount \
  --policy-type TargetTrackingScaling \
  --target-tracking-scaling-policy-configuration '{
    "TargetValue": 70,
    "PredefinedMetricSpecification": {
      "PredefinedMetricType": "ECSServiceAverageCPUUtilization"
    }
  }'
```

## Nomad Job Spec

```hcl
job "api" {
  datacenters = ["dc1"]
  type        = "service"

  group "api" {
    count = 3

    network {
      port "http" { to = 8080 }
    }

    task "api" {
      driver = "docker"

      config {
        image = "registry.example.com/api:latest"
        ports = ["http"]
      }

      resources {
        cpu    = 500  # MHz
        memory = 512  # MB
      }

      env {
        APP_ENV   = "production"
        LOG_LEVEL = "info"
      }

      template {
        data        = <<EOF
DATABASE_URL={{ with secret "secret/prod/api/db" }}{{ .Data.data.url }}{{ end }}
EOF
        destination = "secrets/app.env"
        env         = true
      }

      service {
        name = "api"
        port = "http"
        tags = ["traefik.enable=true"]

        check {
          type     = "http"
          path     = "/healthz"
          interval = "10s"
          timeout  = "2s"
        }
      }
    }
  }
}
```

## Key Rules

- Use `awsvpc` network mode for Fargate — each task gets its own ENI and security group.
- Separate execution role (ECR/Secrets access) from task role (app permissions).
- Use `secrets` in task definition to inject Secrets Manager values at launch — never bake secrets into images.
- Set `minimumHealthyPercent: 100` for zero-downtime rolling deploys.
- Nomad `template` with Vault integration handles secrets without baking them into job specs.
- Nomad services register with Consul automatically when Consul is configured as the service discovery backend.
