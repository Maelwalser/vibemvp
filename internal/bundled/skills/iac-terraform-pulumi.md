# Terraform & Pulumi Skill Guide

## Overview

Infrastructure-as-Code tools provision and manage cloud resources declaratively. Terraform uses HCL; Pulumi uses general-purpose languages (TypeScript, Python, Go).

## Terraform — Core Structure

```hcl
# main.tf
terraform {
  required_version = ">= 1.6"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket         = "my-tfstate"
    key            = "api/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "terraform-locks"   # Distributed lock
    encrypt        = true
  }
}

provider "aws" {
  region = var.aws_region
}
```

```hcl
# variables.tf
variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "environment" {
  type    = string
  description = "Deployment environment: dev, staging, prod"
  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Must be dev, staging, or prod."
  }
}

variable "db_password" {
  type      = string
  sensitive = true
}
```

```hcl
# resources.tf
resource "aws_instance" "api" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = var.environment == "prod" ? "t3.medium" : "t3.micro"

  tags = {
    Name        = "api-${var.environment}"
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"]   # Canonical
  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-22.04-amd64-server-*"]
  }
}
```

```hcl
# outputs.tf
output "api_public_ip" {
  value       = aws_instance.api.public_ip
  description = "Public IP of the API server"
}
```

## Remote State Backend (S3 + DynamoDB)

```bash
# Create state bucket and lock table (one-time setup)
aws s3api create-bucket --bucket my-tfstate --region us-east-1
aws s3api put-bucket-versioning \
  --bucket my-tfstate \
  --versioning-configuration Status=Enabled
aws dynamodb create-table \
  --table-name terraform-locks \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST
```

## Workspace-Based Environment Separation

```bash
terraform workspace new dev
terraform workspace new staging
terraform workspace new prod

terraform workspace select prod
terraform plan -var="environment=prod" -var-file="prod.tfvars"
```

```hcl
# Use workspace in resource names
locals {
  env = terraform.workspace
}

resource "aws_s3_bucket" "assets" {
  bucket = "myapp-${local.env}-assets"
}
```

## Import Existing Resources

```bash
# Import existing resource into state
terraform import aws_s3_bucket.existing my-existing-bucket

# Terraform 1.5+ — import block in config
import {
  to = aws_s3_bucket.existing
  id = "my-existing-bucket"
}
```

## Safe Destroy Workflow

```bash
# Always review plan before destroying
terraform plan -destroy -out=destroy.tfplan
# Review destroy.tfplan carefully
terraform apply destroy.tfplan
```

## Module Structure

```
modules/
├── vpc/
│   ├── main.tf
│   ├── variables.tf
│   └── outputs.tf
└── rds/
    ├── main.tf
    ├── variables.tf
    └── outputs.tf
```

```hcl
module "vpc" {
  source = "./modules/vpc"
  cidr   = "10.0.0.0/16"
  env    = var.environment
}

module "database" {
  source    = "./modules/rds"
  vpc_id    = module.vpc.id
  subnet_ids = module.vpc.private_subnet_ids
  env       = var.environment
}
```

## Pulumi — TypeScript Stack

```typescript
// index.ts
import * as aws from "@pulumi/aws";
import * as pulumi from "@pulumi/pulumi";

const config = new pulumi.Config();
const env = config.require("environment");
const dbPassword = config.requireSecret("dbPassword");

const bucket = new aws.s3.Bucket("assets", {
  bucket: `myapp-${env}-assets`,
  tags: { Environment: env, ManagedBy: "pulumi" },
});

const db = new aws.rds.Instance("postgres", {
  engine: "postgres",
  engineVersion: "16",
  instanceClass: env === "prod" ? "db.t3.medium" : "db.t3.micro",
  allocatedStorage: 20,
  dbName: "appdb",
  username: "admin",
  password: dbPassword,
  skipFinalSnapshot: env !== "prod",
});

// Stack outputs
export const bucketName = bucket.id;
export const dbEndpoint = db.endpoint;
```

```bash
# Pulumi CLI
pulumi stack init dev
pulumi config set environment dev
pulumi config set --secret dbPassword supersecret
pulumi up --yes
pulumi stack output dbEndpoint

# Import existing resource
pulumi import aws:s3/bucket:Bucket existing my-existing-bucket
```

## Key Rules

- Always use remote state with locking (S3+DynamoDB or Terraform Cloud) — never local state in CI.
- Workspaces or separate state files per environment prevent accidental cross-env changes.
- Mark sensitive variables with `sensitive = true` — Terraform will redact them in output.
- Run `terraform plan -destroy` and review before any destructive apply.
- Use `terraform fmt` and `terraform validate` in CI before plan/apply.
- Pulumi `Config.requireSecret()` stores values encrypted in the state file.
- Never hardcode credentials — use environment variables (`AWS_ACCESS_KEY_ID`) or OIDC federation.
