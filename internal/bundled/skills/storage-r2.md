# Cloudflare R2 Skill Guide

## Overview

Cloudflare R2 is S3-compatible object storage with zero egress fees. It supports:
- S3-compatible API (same AWS SDK, custom endpoint)
- Cloudflare Workers native binding
- Custom domain + CDN integration

## Endpoint Format

```
https://<ACCOUNT_ID>.r2.cloudflarestorage.com
```

Find your Account ID in the Cloudflare dashboard → right sidebar.

## AWS SDK with Custom Endpoint

```typescript
import { S3Client, PutObjectCommand, GetObjectCommand, DeleteObjectCommand } from "@aws-sdk/client-s3";

const r2 = new S3Client({
  region: "auto",   // R2 uses "auto" as the region
  endpoint: `https://${process.env.CF_ACCOUNT_ID}.r2.cloudflarestorage.com`,
  credentials: {
    accessKeyId: process.env.R2_ACCESS_KEY_ID!,
    secretAccessKey: process.env.R2_SECRET_ACCESS_KEY!,
  },
});
```

```python
import boto3

r2 = boto3.client(
    "s3",
    endpoint_url=f"https://{os.environ['CF_ACCOUNT_ID']}.r2.cloudflarestorage.com",
    aws_access_key_id=os.environ["R2_ACCESS_KEY_ID"],
    aws_secret_access_key=os.environ["R2_SECRET_ACCESS_KEY"],
    region_name="auto",
)
```

```go
import (
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/credentials"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", os.Getenv("CF_ACCOUNT_ID"))

cfg, _ := config.LoadDefaultConfig(context.TODO(),
    config.WithRegion("auto"),
    config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
        os.Getenv("R2_ACCESS_KEY_ID"),
        os.Getenv("R2_SECRET_ACCESS_KEY"),
        "",
    )),
)

r2Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
    o.BaseEndpoint = aws.String(endpoint)
})
```

## Create R2 API Token

Cloudflare Dashboard → R2 → Manage R2 API Tokens → Create API Token.

Permissions: `Object Read & Write` (or read-only for download-only workers).

## Cloudflare Workers Binding

```toml
# wrangler.toml
name = "my-worker"
main = "src/index.ts"
compatibility_date = "2024-01-01"

[[r2_buckets]]
binding = "MY_BUCKET"
bucket_name = "my-production-bucket"

# Dev binding (local bucket)
[[r2_buckets]]
binding = "MY_BUCKET"
bucket_name = "my-dev-bucket"
preview_bucket_name = "my-dev-bucket"
```

```typescript
// src/index.ts
export interface Env {
  MY_BUCKET: R2Bucket;
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url);
    const key = url.pathname.slice(1);

    switch (request.method) {
      case "PUT": {
        await env.MY_BUCKET.put(key, request.body, {
          httpMetadata: { contentType: request.headers.get("Content-Type") ?? "application/octet-stream" },
          customMetadata: { uploadedAt: new Date().toISOString() },
        });
        return new Response(`Uploaded ${key}`, { status: 200 });
      }

      case "GET": {
        const object = await env.MY_BUCKET.get(key);
        if (!object) return new Response("Not Found", { status: 404 });

        const headers = new Headers();
        object.writeHttpMetadata(headers);
        headers.set("etag", object.httpEtag);
        headers.set("Cache-Control", "public, max-age=31536000, immutable");

        return new Response(object.body, { headers });
      }

      case "DELETE": {
        await env.MY_BUCKET.delete(key);
        return new Response("Deleted", { status: 200 });
      }

      default:
        return new Response("Method Not Allowed", { status: 405 });
    }
  },
};
```

## Presigned Upload URL from Worker

```typescript
import { AwsClient } from "aws4fetch";

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    if (request.method !== "POST" || new URL(request.url).pathname !== "/presign") {
      return new Response("Not Found", { status: 404 });
    }

    const { filename, contentType } = await request.json<{
      filename: string;
      contentType: string;
    }>();

    const r2 = new AwsClient({
      accessKeyId: env.R2_ACCESS_KEY_ID,
      secretAccessKey: env.R2_SECRET_ACCESS_KEY,
      region: "auto",
      service: "s3",
    });

    const url = new URL(
      `https://${env.CF_ACCOUNT_ID}.r2.cloudflarestorage.com/${env.BUCKET_NAME}/${filename}`
    );
    url.searchParams.set("X-Amz-Expires", "900");

    const signed = await r2.sign(
      new Request(url, { method: "PUT" }),
      { aws: { signQuery: true } }
    );

    return Response.json({ url: signed.url });
  },
};
```

## CORS Configuration

Configure in Cloudflare Dashboard → R2 → Bucket → Settings → CORS Policy:

```json
[
  {
    "AllowedOrigins": ["https://app.example.com"],
    "AllowedMethods": ["GET", "PUT", "POST"],
    "AllowedHeaders": ["*"],
    "ExposeHeaders": ["ETag"],
    "MaxAgeSeconds": 3000
  }
]
```

Or via AWS SDK:

```python
r2.put_bucket_cors(
    Bucket="my-bucket",
    CORSConfiguration={
        "CORSRules": [{
            "AllowedOrigins": ["https://app.example.com"],
            "AllowedMethods": ["GET", "PUT"],
            "AllowedHeaders": ["*"],
            "MaxAgeSeconds": 3000,
        }]
    },
)
```

## Custom Domain + Cache-Control

1. Add a custom domain in R2 bucket settings → Custom Domains → Connect Domain.
2. Cloudflare proxies the domain (orange cloud).
3. Objects are served through Cloudflare CDN.

```typescript
// Set Cache-Control when uploading for CDN caching
await env.MY_BUCKET.put(key, body, {
  httpMetadata: {
    contentType: "image/webp",
    cacheControl: "public, max-age=31536000, immutable",  // 1 year for immutable assets
    // For mutable assets:
    // cacheControl: "public, max-age=3600, stale-while-revalidate=86400",
  },
});
```

## Bucket Operations

```typescript
// List objects
const listed = await env.MY_BUCKET.list({
  prefix: "uploads/",
  limit: 100,
  cursor: request.url.searchParams.get("cursor") ?? undefined,
});

return Response.json({
  objects: listed.objects.map((o) => ({ key: o.key, size: o.size })),
  truncated: listed.truncated,
  cursor: listed.cursor,
});
```

## Multipart Upload (Large Files)

```python
# R2 supports multipart via S3 API
mpu = r2.create_multipart_upload(Bucket="my-bucket", Key="large-file.zip")
upload_id = mpu["UploadId"]

parts = []
with open("large-file.zip", "rb") as f:
    for i, chunk in enumerate(iter(lambda: f.read(10 * 1024 * 1024), b""), 1):
        part = r2.upload_part(
            Bucket="my-bucket",
            Key="large-file.zip",
            UploadId=upload_id,
            PartNumber=i,
            Body=chunk,
        )
        parts.append({"PartNumber": i, "ETag": part["ETag"]})

r2.complete_multipart_upload(
    Bucket="my-bucket",
    Key="large-file.zip",
    UploadId=upload_id,
    MultipartUpload={"Parts": parts},
)
```

## Anti-Patterns

- R2 does not charge egress — but inter-region egress within Cloudflare Workers is still free only within same datacenter. Design your Worker to be in the same location as the bucket.
- Do not use `forcePathStyle` with R2 — it uses virtual-hosted style by default at the account endpoint.
- Do not store R2 credentials in Workers code — use bindings for native access, or Secrets for API token–based access.
- Presigned URLs expire — do not cache them on the client longer than their expiry.
- Always set `Cache-Control` on immutable assets (versioned URLs) — default CDN TTL is short.
