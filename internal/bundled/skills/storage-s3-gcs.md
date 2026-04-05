# AWS S3 & Google Cloud Storage Skill Guide

---

## AWS S3

### Installation

```bash
pip install boto3
# or
npm install @aws-sdk/client-s3 @aws-sdk/s3-request-presigner
go get github.com/aws/aws-sdk-go-v2/service/s3
```

### Client Setup

```python
import boto3

s3 = boto3.client(
    "s3",
    region_name=os.environ["AWS_REGION"],
    # Credentials from env: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
    # Or use IAM role (preferred in production)
)
```

```typescript
import { S3Client } from "@aws-sdk/client-s3";

const s3 = new S3Client({
  region: process.env.AWS_REGION,
  // Credentials from env or IAM role automatically
});
```

### PutObject / GetObject / DeleteObject

```python
# Upload
s3.put_object(
    Bucket="my-bucket",
    Key="uploads/image.jpg",
    Body=file_bytes,
    ContentType="image/jpeg",
)

# Download
response = s3.get_object(Bucket="my-bucket", Key="uploads/image.jpg")
data = response["Body"].read()

# Delete
s3.delete_object(Bucket="my-bucket", Key="uploads/image.jpg")

# Delete multiple objects
s3.delete_objects(
    Bucket="my-bucket",
    Delete={"Objects": [{"Key": "uploads/a.jpg"}, {"Key": "uploads/b.jpg"}]},
)
```

### Presigned URLs

```python
from botocore.exceptions import ClientError

def create_download_url(bucket: str, key: str, expires_in: int = 900) -> str:
    """Generate a presigned URL for reading an object (15 min default)."""
    return s3.generate_presigned_url(
        "get_object",
        Params={"Bucket": bucket, "Key": key},
        ExpiresIn=expires_in,
    )

def create_upload_url(bucket: str, key: str, content_type: str, max_size_mb: int = 10) -> dict:
    """Generate a presigned POST URL for client-side upload."""
    return s3.generate_presigned_post(
        bucket,
        key,
        Fields={"Content-Type": content_type},
        Conditions=[
            {"Content-Type": content_type},
            ["content-length-range", 1, max_size_mb * 1024 * 1024],
        ],
        ExpiresIn=900,
    )
```

```typescript
import { GetObjectCommand, PutObjectCommand } from "@aws-sdk/client-s3";
import { getSignedUrl } from "@aws-sdk/s3-request-presigner";

// Download presigned URL
const downloadUrl = await getSignedUrl(
  s3,
  new GetObjectCommand({ Bucket: "my-bucket", Key: "file.pdf" }),
  { expiresIn: 900 }
);

// Upload presigned URL (client uploads directly to S3)
const uploadUrl = await getSignedUrl(
  s3,
  new PutObjectCommand({
    Bucket: "my-bucket",
    Key: "uploads/file.pdf",
    ContentType: "application/pdf",
  }),
  { expiresIn: 900 }
);
```

### Multipart Upload (>100 MB)

```python
import boto3
from boto3.s3.transfer import TransferConfig

config = TransferConfig(
    multipart_threshold=100 * 1024 * 1024,  # 100 MB
    multipart_chunksize=10 * 1024 * 1024,   # 10 MB chunks
    max_concurrency=10,
    use_threads=True,
)

s3.upload_file(
    Filename="/path/to/large-file.zip",
    Bucket="my-bucket",
    Key="large-file.zip",
    Config=config,
    Callback=lambda bytes_transferred: print(f"Uploaded {bytes_transferred} bytes"),
)
```

### Bucket Policy (JSON)

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "AllowPublicRead",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::my-public-bucket/public/*"
    },
    {
      "Sid": "DenyNonHTTPS",
      "Effect": "Deny",
      "Principal": "*",
      "Action": "s3:*",
      "Resource": [
        "arn:aws:s3:::my-bucket",
        "arn:aws:s3:::my-bucket/*"
      ],
      "Condition": {
        "Bool": { "aws:SecureTransport": "false" }
      }
    }
  ]
}
```

### CORS Configuration

```xml
<!-- Applied via PutBucketCors API -->
<CORSConfiguration>
  <CORSRule>
    <AllowedOrigin>https://app.example.com</AllowedOrigin>
    <AllowedMethod>GET</AllowedMethod>
    <AllowedMethod>PUT</AllowedMethod>
    <AllowedMethod>POST</AllowedMethod>
    <AllowedHeader>*</AllowedHeader>
    <MaxAgeSeconds>3000</MaxAgeSeconds>
    <ExposeHeader>ETag</ExposeHeader>
  </CORSRule>
</CORSConfiguration>
```

### Lifecycle Rules

```python
s3.put_bucket_lifecycle_configuration(
    Bucket="my-bucket",
    LifecycleConfiguration={
        "Rules": [
            {
                "ID": "transition-to-ia",
                "Status": "Enabled",
                "Filter": {"Prefix": "uploads/"},
                "Transitions": [
                    {"Days": 30, "StorageClass": "STANDARD_IA"},
                    {"Days": 90, "StorageClass": "GLACIER"},
                ],
                "Expiration": {"Days": 365},
            }
        ]
    },
)
```

### Content Type Enforcement (Lambda@Edge)

```javascript
// CloudFront Lambda@Edge: viewer-request
exports.handler = async (event) => {
  const request = event.Records[0].cf.request;
  const contentType = request.headers["content-type"]?.[0]?.value || "";
  const allowed = ["image/jpeg", "image/png", "image/webp", "application/pdf"];

  if (request.method === "PUT" && !allowed.includes(contentType)) {
    return {
      status: "415",
      statusDescription: "Unsupported Media Type",
      body: JSON.stringify({ error: "Content type not allowed" }),
    };
  }
  return request;
};
```

---

## Google Cloud Storage (GCS)

### Installation

```bash
pip install google-cloud-storage
npm install @google-cloud/storage
```

### Client Setup

```python
from google.cloud import storage

client = storage.Client(project=os.environ["GCP_PROJECT_ID"])
bucket = client.bucket("my-bucket")
```

### Upload / Download / Delete

```python
# Upload
blob = bucket.blob("uploads/image.jpg")
blob.upload_from_file(file_obj, content_type="image/jpeg")
# or from filename:
blob.upload_from_filename("/tmp/image.jpg")

# Download
blob.download_to_filename("/tmp/downloaded.jpg")
data = blob.download_as_bytes()

# Delete
blob.delete()
```

### Presigned (Signed) URLs

```python
from datetime import timedelta

# Download URL (15 min)
signed_url = blob.generate_signed_url(
    expiration=timedelta(minutes=15),
    method="GET",
    version="v4",
)

# Upload URL
upload_url = blob.generate_signed_url(
    expiration=timedelta(minutes=15),
    method="PUT",
    content_type="image/jpeg",
    version="v4",
)
```

### Lifecycle Policy (GCS)

```python
from google.cloud.storage import Bucket

bucket = client.bucket("my-bucket")
rules = [
    {
        "action": {"type": "SetStorageClass", "storageClass": "NEARLINE"},
        "condition": {"age": 30},
    },
    {
        "action": {"type": "SetStorageClass", "storageClass": "COLDLINE"},
        "condition": {"age": 90},
    },
    {
        "action": {"type": "Delete"},
        "condition": {"age": 365},
    },
]
bucket.lifecycle_rules = rules
bucket.patch()
```

## Anti-Patterns

- Never store AWS credentials in code — use IAM roles or environment variables.
- Do not use presigned URLs for long-lived (>7 days) access — use CloudFront signed cookies.
- Always enforce HTTPS-only via bucket policy for production buckets.
- Avoid uploading large files through your server — use presigned PUT URLs for direct client upload.
- Do not set lifecycle rules on the entire bucket root if different prefixes need different policies — use prefix filters.
