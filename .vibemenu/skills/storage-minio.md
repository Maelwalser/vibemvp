# MinIO Skill Guide

## Overview

MinIO is S3-compatible object storage. Applications can use AWS SDK with a custom endpoint to talk to MinIO, making it a drop-in replacement for S3 in development and self-hosted deployments.

## Deployment Modes

### Single-Node (Development / Small Production)

```bash
# Docker
docker run -d \
  -p 9000:9000 \
  -p 9001:9001 \
  -v /data/minio:/data \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin123 \
  quay.io/minio/minio server /data --console-address ":9001"
```

### Distributed with Erasure Coding (Production)

Minimum: 4 drives. Erasure coding provides redundancy — data survives up to N/2 drive failures.

```bash
# 4-node, 4-drive-per-node setup (16 drives total)
# On each node, run:
minio server \
  http://minio{1...4}/data{1...4} \
  --console-address ":9001"
```

```yaml
# docker-compose.yml (4-node example)
version: "3.8"
services:
  minio1:
    image: quay.io/minio/minio
    command: server --console-address ":9001" http://minio{1...4}/data{1...2}
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - minio1-data1:/data1
      - minio1-data2:/data2
    # ... repeat for minio2, minio3, minio4
```

Erasure coding setup: `MINIO_ERASURE_SET_DRIVE_COUNT` controls parity. With 16 drives, default parity = 4 (EC:4).

## S3-Compatible API (Same SDK)

```python
import boto3

s3 = boto3.client(
    "s3",
    endpoint_url="http://localhost:9000",
    aws_access_key_id="minioadmin",
    aws_secret_access_key="minioadmin123",
    region_name="us-east-1",  # MinIO ignores region but SDK requires it
)
```

```typescript
import { S3Client } from "@aws-sdk/client-s3";

const s3 = new S3Client({
  endpoint: "http://localhost:9000",
  region: "us-east-1",
  credentials: {
    accessKeyId: process.env.MINIO_ACCESS_KEY!,
    secretAccessKey: process.env.MINIO_SECRET_KEY!,
  },
  forcePathStyle: true,   // REQUIRED for MinIO — use path-style addressing
});
```

```go
import (
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

cfg, _ := config.LoadDefaultConfig(context.TODO(),
    config.WithRegion("us-east-1"),
    config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
        os.Getenv("MINIO_ACCESS_KEY"),
        os.Getenv("MINIO_SECRET_KEY"),
        "",
    )),
)

client := s3.NewFromConfig(cfg, func(o *s3.Options) {
    o.BaseEndpoint = aws.String("http://localhost:9000")
    o.UsePathStyle = true   // Required for MinIO
})
```

## MinIO Go SDK (Alternative to AWS SDK)

```go
import "github.com/minio/minio-go/v7"

minioClient, err := minio.New("localhost:9000", &minio.Options{
    Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
    Secure: false, // true for HTTPS
})

// Create bucket
err = minioClient.MakeBucket(ctx, "my-bucket", minio.MakeBucketOptions{Region: "us-east-1"})

// Upload
_, err = minioClient.FPutObject(ctx, "my-bucket", "key", "/tmp/file.jpg",
    minio.PutObjectOptions{ContentType: "image/jpeg"})

// Download
obj, err := minioClient.GetObject(ctx, "my-bucket", "key", minio.GetObjectOptions{})
```

## mc (MinIO Client) CLI

```bash
# Configure alias
mc alias set local http://localhost:9000 minioadmin minioadmin123

# Create bucket
mc mb local/my-bucket

# Upload
mc cp /path/to/file.jpg local/my-bucket/uploads/file.jpg

# List
mc ls local/my-bucket

# Presigned URL (download, 1 hour)
mc share download local/my-bucket/uploads/file.jpg --expire 1h

# Presigned URL (upload)
mc share upload local/my-bucket/uploads/newfile.jpg --expire 1h
```

## Bucket Lifecycle Policy (XML)

```bash
mc ilm import local/my-bucket <<'EOF'
{
  "Rules": [
    {
      "ID": "expire-old-uploads",
      "Status": "Enabled",
      "Filter": { "Prefix": "uploads/" },
      "Expiration": { "Days": 90 }
    },
    {
      "ID": "transition-logs",
      "Status": "Enabled",
      "Filter": { "Prefix": "logs/" },
      "Transition": { "Days": 30, "StorageClass": "GLACIER" }
    }
  ]
}
EOF
```

## IAM Policy with mc

```bash
# Create policy file
cat > read-write-policy.json <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": ["s3:GetObject", "s3:PutObject", "s3:DeleteObject"],
      "Resource": "arn:aws:s3:::my-bucket/*"
    },
    {
      "Effect": "Allow",
      "Action": "s3:ListBucket",
      "Resource": "arn:aws:s3:::my-bucket"
    }
  ]
}
EOF

mc admin policy add local app-policy read-write-policy.json
mc admin user add local app-user app-password
mc admin policy attach local app-policy --user app-user
```

## TLS with Custom Certificate

```bash
# Place certs in MinIO's config directory
mkdir -p /data/minio/.minio/certs
cp server.crt /data/minio/.minio/certs/public.crt
cp server.key /data/minio/.minio/certs/private.key

# Start with HTTPS
minio server --address ":443" /data
```

For self-signed certs, place CA cert at `/data/minio/.minio/certs/CAs/ca.crt`.

## Performance Tuning for NVMe

```bash
# Direct I/O bypasses OS page cache — better for NVMe
export MINIO_DIRECT_IO="on"

# Increase read/write concurrency
export MINIO_API_REQUESTS_MAX="600"
export MINIO_API_REQUESTS_DEADLINE="10s"

# On Linux: set I/O scheduler for NVMe drives
echo "none" > /sys/block/nvme0n1/queue/scheduler
```

Environment variables:

```bash
# Increase object parts for multipart (default 10000 parts max)
MINIO_STORAGE_CLASS_STANDARD=EC:2   # Parity shards
MINIO_STORAGE_CLASS_RRS=EC:1        # Reduced redundancy
```

## Presigned URL (Python SDK)

```python
from botocore.config import Config

s3 = boto3.client(
    "s3",
    endpoint_url="http://localhost:9000",
    aws_access_key_id="minioadmin",
    aws_secret_access_key="minioadmin123",
    region_name="us-east-1",
    config=Config(signature_version="s3v4"),   # Required for MinIO
)

url = s3.generate_presigned_url(
    "get_object",
    Params={"Bucket": "my-bucket", "Key": "uploads/file.jpg"},
    ExpiresIn=3600,
)
```

## Health Check

```bash
curl -I http://localhost:9000/minio/health/live
# HTTP 200 = healthy

curl -I http://localhost:9000/minio/health/cluster
# HTTP 200 = cluster quorum OK
```

## Anti-Patterns

- Always set `forcePathStyle: true` (AWS SDK) or `UsePathStyle: true` — MinIO does not support virtual-hosted-style by default.
- Do not run single-node MinIO without a persistent volume — restart loses all data.
- Distributed mode requires an odd number of drives per erasure set for best parity.
- Do not mix drive types (HDD/SSD/NVMe) in the same erasure set — slowest drive bottlenecks all.
- Never use root credentials in application code — create dedicated users with scoped IAM policies.
