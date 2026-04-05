# Infrastructure Backup & Disaster Recovery Skill Guide

## Overview

Cross-region replication, automated snapshots, point-in-time recovery, and failover runbooks. Target RTO/RPO defined per SLA tier.

## RTO / RPO Targets

| SLA Tier | Uptime | RTO | RPO |
|----------|--------|-----|-----|
| Standard | 99.9% | 1 hour | 1 hour |
| Enhanced | 99.95% | 15 min | 15 min |
| Premium | 99.99% | 5 min | 1 min |

## RDS Automated Backups with Cross-Region Copy

```hcl
resource "aws_db_instance" "primary" {
  identifier              = "myapp-primary"
  engine                  = "postgres"
  engine_version          = "16.2"
  instance_class          = "db.t3.medium"
  multi_az                = true
  backup_retention_period = 30
  backup_window           = "03:00-04:00"
  maintenance_window      = "mon:04:00-mon:05:00"
  deletion_protection     = true

  # Enable automated backups
  skip_final_snapshot = false
  final_snapshot_identifier = "myapp-final-snapshot"
}

resource "aws_db_instance_automated_backups_replication" "cross_region" {
  source_db_instance_arn = aws_db_instance.primary.arn
  retention_period       = 30
  provider               = aws.us-west-2  # secondary region
}
```

## RDS Multi-AZ + Aurora Global Database

```hcl
# Aurora Global Database — RPO < 1 second
resource "aws_rds_global_cluster" "global" {
  global_cluster_identifier = "myapp-global"
  engine                    = "aurora-postgresql"
  engine_version            = "16.2"
  database_name             = "myapp"
}

resource "aws_rds_cluster" "primary" {
  cluster_identifier        = "myapp-primary"
  global_cluster_identifier = aws_rds_global_cluster.global.id
  engine                    = "aurora-postgresql"
  engine_version            = "16.2"
  master_username           = var.db_user
  master_password           = var.db_password
  backup_retention_period   = 30
  preferred_backup_window   = "03:00-04:00"
}

resource "aws_rds_cluster" "secondary" {
  provider                  = aws.eu-west-1
  cluster_identifier        = "myapp-secondary"
  global_cluster_identifier = aws_rds_global_cluster.global.id
  engine                    = "aurora-postgresql"
  engine_version            = "16.2"
}
```

## Point-in-Time Recovery (PITR)

```hcl
# RDS PITR: retain 7–35 days
resource "aws_db_instance" "app" {
  backup_retention_period = 14   # days; min 7, max 35
}

# Restore to specific point in time (CLI)
# aws rds restore-db-instance-to-point-in-time \
#   --source-db-instance-identifier myapp-primary \
#   --target-db-instance-identifier myapp-pitr-restore \
#   --restore-time 2024-01-15T12:00:00Z
```

## S3 Cross-Region Replication (CRR)

```hcl
resource "aws_s3_bucket_replication_configuration" "crr" {
  role   = aws_iam_role.replication.arn
  bucket = aws_s3_bucket.primary.id

  rule {
    id     = "replicate-all"
    status = "Enabled"

    destination {
      bucket        = aws_s3_bucket.replica.arn
      storage_class = "STANDARD_IA"
    }

    delete_marker_replication {
      status = "Enabled"
    }
  }
}

resource "aws_iam_role" "replication" {
  name = "s3-crr-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action    = "sts:AssumeRole"
      Effect    = "Allow"
      Principal = { Service = "s3.amazonaws.com" }
    }]
  })
}
```

## DynamoDB Global Tables

```hcl
resource "aws_dynamodb_table" "app" {
  name             = "myapp-table"
  billing_mode     = "PAY_PER_REQUEST"
  hash_key         = "pk"
  range_key        = "sk"
  stream_enabled   = true
  stream_view_type = "NEW_AND_OLD_IMAGES"

  attribute {
    name = "pk"
    type = "S"
  }
  attribute {
    name = "sk"
    type = "S"
  }

  replica {
    region_name = "eu-west-1"
  }
  replica {
    region_name = "ap-southeast-1"
  }

  point_in_time_recovery {
    enabled = true
  }
}
```

## AWS Backup Vault (Centralized Policy)

```hcl
resource "aws_backup_plan" "daily" {
  name = "daily-backup-plan"

  rule {
    rule_name         = "daily-30-day-retention"
    target_vault_name = aws_backup_vault.primary.name
    schedule          = "cron(0 3 * * ? *)"  # 3am UTC daily

    lifecycle {
      delete_after = 30
    }

    copy_action {
      destination_vault_arn = aws_backup_vault.secondary.arn
    }
  }
}

resource "aws_backup_selection" "all" {
  name         = "all-resources"
  plan_id      = aws_backup_plan.daily.id
  iam_role_arn = aws_iam_role.backup.arn

  resources = ["*"]

  condition {
    string_equals {
      key   = "aws:ResourceTag/backup"
      value = "true"
    }
  }
}
```

## Backup Verification (Weekly Restore to Staging)

```bash
#!/usr/bin/env bash
# verify-backup.sh — run weekly via cron/CI
set -euo pipefail

SNAPSHOT=$(aws rds describe-db-snapshots \
  --db-instance-identifier myapp-primary \
  --query 'DBSnapshots | sort_by(@, &SnapshotCreateTime) | [-1].DBSnapshotIdentifier' \
  --output text)

echo "Restoring snapshot: $SNAPSHOT"

aws rds restore-db-instance-from-db-snapshot \
  --db-instance-identifier myapp-verify-$(date +%Y%m%d) \
  --db-snapshot-identifier "$SNAPSHOT" \
  --db-instance-class db.t3.medium \
  --no-multi-az

aws rds wait db-instance-available \
  --db-instance-identifier myapp-verify-$(date +%Y%m%d)

echo "Running smoke queries against restored instance..."
# ... run SQL smoke tests ...

aws rds delete-db-instance \
  --db-instance-identifier myapp-verify-$(date +%Y%m%d) \
  --skip-final-snapshot

echo "Backup verification complete"
```

## Failover Runbook (DNS Cutover)

```markdown
### Failover Steps (Standard tier — RTO 1h)

1. **Declare incident** — alert on-call, open incident channel.
2. **Verify primary outage** — confirm primary region health dashboard.
3. **Promote Aurora secondary**
   aws rds failover-global-cluster \
     --global-cluster-identifier myapp-global \
     --target-db-cluster-identifier myapp-secondary
4. **Update Route53 DNS** — point CNAME/A record to secondary endpoint.
   aws route53 change-resource-record-sets \
     --hosted-zone-id ZXXXXX \
     --change-batch file://dns-failover.json
5. **Verify connectivity** — run readiness probe against /readyz.
6. **Notify stakeholders** — status page update.
7. **Monitor for 30 min** — check error rate and latency.
8. **Post-incident review** — document RTO achieved vs target.
```

## Key Rules

- Always set `backup_retention_period >= 7` for any production RDS instance.
- Enable `multi_az = true` for Standard tier and above.
- DynamoDB global tables require `stream_enabled = true` with `NEW_AND_OLD_IMAGES`.
- PITR must be enabled alongside automated backups — they are separate features.
- Test restores weekly in staging; an untested backup is not a backup.
- Tag all production resources with `backup = "true"` to auto-enroll in centralized vault.
- Aurora Global Database achieves RPO < 1 second; standard Multi-AZ achieves RPO ~1 minute.
- Cross-region S3 replication does not replicate existing objects — seed the replica bucket on creation.
