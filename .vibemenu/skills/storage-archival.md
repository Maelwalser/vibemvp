# Archival Storage Skill Guide

## Storage Tier Comparison

| Provider | Hot/Standard | Warm/IA | Cold/Archive | Deep Archive |
|----------|-------------|---------|-------------|-------------|
| AWS S3 | STANDARD | STANDARD_IA, ONE_ZONE_IA | GLACIER_IR, GLACIER | DEEP_ARCHIVE |
| GCS | Standard | Nearline (30d min) | Coldline (90d min) | Archive (365d min) |
| Azure | Hot | Cool | Cold | Archive |

Access cost increases and storage cost decreases as you move to colder tiers.

---

## AWS S3 Glacier

### Lifecycle Rule: STANDARD → STANDARD_IA → GLACIER → DEEP_ARCHIVE

```python
import boto3

s3 = boto3.client("s3", region_name=os.environ["AWS_REGION"])

s3.put_bucket_lifecycle_configuration(
    Bucket="my-data-bucket",
    LifecycleConfiguration={
        "Rules": [
            {
                "ID": "archive-uploads",
                "Status": "Enabled",
                "Filter": {"Prefix": "uploads/"},
                "Transitions": [
                    {"Days": 30,  "StorageClass": "STANDARD_IA"},
                    {"Days": 90,  "StorageClass": "GLACIER"},
                    {"Days": 180, "StorageClass": "DEEP_ARCHIVE"},
                ],
                "Expiration": {"Days": 2555},  # 7 years
                "NoncurrentVersionTransitions": [
                    {"NoncurrentDays": 30, "StorageClass": "GLACIER"},
                ],
                "NoncurrentVersionExpiration": {"NoncurrentDays": 365},
            },
            {
                "ID": "abort-incomplete-multipart",
                "Status": "Enabled",
                "Filter": {},
                "AbortIncompleteMultipartUpload": {"DaysAfterInitiation": 7},
            },
        ]
    },
)
```

### Restore from Glacier

```python
# Glacier: must restore before downloading (1–12 hours standard, 1–5 min expedited)
s3.restore_object(
    Bucket="my-data-bucket",
    Key="uploads/archive-2022/report.pdf",
    RestoreRequest={
        "Days": 7,  # How many days the restored copy stays available
        "GlacierJobParameters": {
            "Tier": "Standard",  # Expedited | Standard | Bulk
            # Expedited: 1–5 min, most expensive
            # Standard:  3–5 hours, moderate cost
            # Bulk:      5–12 hours, cheapest
        },
    },
)

# Check restore status
response = s3.head_object(Bucket="my-data-bucket", Key="uploads/archive-2022/report.pdf")
restore_status = response.get("Restore", "")
# restore_status: 'ongoing-request="true"' (in progress)
# restore_status: 'ongoing-request="false", expiry-date="Mon, 22 Jan 2024 00:00:00 GMT"' (done)
```

### S3 Intelligent-Tiering (Auto-Tiering)

```python
# Intelligent-Tiering automatically moves objects between tiers based on access patterns
s3.put_bucket_intelligent_tiering_configuration(
    Bucket="my-data-bucket",
    Id="EntireBucket",
    IntelligentTieringConfiguration={
        "Id": "EntireBucket",
        "Status": "Enabled",
        "Tierings": [
            {"Days": 90,  "AccessTier": "ARCHIVE_ACCESS"},
            {"Days": 180, "AccessTier": "DEEP_ARCHIVE_ACCESS"},
        ],
    },
)
```

---

## Google Cloud Storage Archive Class

### Auto-Tiering with Storage Class

```python
from google.cloud import storage

client = storage.Client(project=os.environ["GCP_PROJECT_ID"])
bucket = client.bucket("my-archive-bucket")

# Set lifecycle rules for auto-tiering
rules = [
    {
        "action": {"type": "SetStorageClass", "storageClass": "NEARLINE"},
        "condition": {"age": 30, "matchesStorageClass": ["STANDARD"]},
    },
    {
        "action": {"type": "SetStorageClass", "storageClass": "COLDLINE"},
        "condition": {"age": 90, "matchesStorageClass": ["NEARLINE"]},
    },
    {
        "action": {"type": "SetStorageClass", "storageClass": "ARCHIVE"},
        "condition": {"age": 365, "matchesStorageClass": ["COLDLINE"]},
    },
    {
        "action": {"type": "Delete"},
        "condition": {"age": 2555},  # 7 years
    },
]

bucket.lifecycle_rules = rules
bucket.patch()
```

### Set Storage Class on Upload

```python
blob = bucket.blob("reports/annual-2022.pdf")
blob.storage_class = "COLDLINE"  # STANDARD | NEARLINE | COLDLINE | ARCHIVE
blob.upload_from_filename("/tmp/annual-2022.pdf")
```

### GCS Restore (Archive → Standard)

Archive class objects must be rewritten to a warmer class before downloading:

```python
blob = bucket.blob("reports/annual-2022.pdf")
blob.update_storage_class("STANDARD")  # Triggers a rewrite; takes a moment
# Then download normally
data = blob.download_as_bytes()
```

---

## Azure Archive Tier

### Lifecycle Policy (Cool → Cold → Archive)

```json
{
  "rules": [
    {
      "name": "archive-policy",
      "enabled": true,
      "type": "Lifecycle",
      "definition": {
        "filters": {
          "blobTypes": ["blockBlob"],
          "prefixMatch": ["backups/", "exports/"]
        },
        "actions": {
          "baseBlob": {
            "tierToCool":    { "daysAfterModificationGreaterThan": 30  },
            "tierToCold":    { "daysAfterModificationGreaterThan": 90  },
            "tierToArchive": { "daysAfterModificationGreaterThan": 180 },
            "delete":        { "daysAfterModificationGreaterThan": 2555 }
          }
        }
      }
    }
  ]
}
```

### Rehydrate from Archive Tier

```python
from azure.storage.blob import BlobServiceClient, RehydratePriority

blob_client = service_client.get_blob_client("backups", "export-2022.parquet")

# Trigger rehydration (1–15 hours for Standard, <1 hour for High)
blob_client.set_blob_tier(
    tier="Cool",
    rehydrate_priority=RehydratePriority.STANDARD,  # or HIGH
)

# Check rehydration status
props = blob_client.get_blob_properties()
print(props.archive_status)  # "rehydrate-pending-to-cool" | None (done)
```

---

## Export Patterns

### Parquet Export (columnar, compressed)

```python
import pyarrow as pa
import pyarrow.parquet as pq
import boto3
import io

def export_to_parquet_s3(data: list[dict], bucket: str, key: str):
    table = pa.Table.from_pylist(data)
    
    buffer = io.BytesIO()
    pq.write_table(
        table,
        buffer,
        compression="snappy",     # snappy | gzip | zstd | brotli
        use_dictionary=True,
        row_group_size=100_000,
    )
    buffer.seek(0)
    
    s3 = boto3.client("s3")
    s3.put_object(
        Bucket=bucket,
        Key=key,
        Body=buffer.getvalue(),
        ContentType="application/parquet",
    )
```

### Gzip CSV Export

```python
import csv
import gzip
import io

def export_to_gzip_csv_s3(rows: list[dict], fieldnames: list[str], bucket: str, key: str):
    buffer = io.BytesIO()
    with gzip.GzipFile(fileobj=buffer, mode="wb") as gz:
        wrapper = io.TextIOWrapper(gz, encoding="utf-8", newline="")
        writer = csv.DictWriter(wrapper, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(rows)
        wrapper.detach()
    
    buffer.seek(0)
    s3 = boto3.client("s3")
    s3.put_object(
        Bucket=bucket,
        Key=key,
        Body=buffer.getvalue(),
        ContentType="text/csv",
        ContentEncoding="gzip",
    )
```

---

## Recovery SLA by Retrieval Tier

| Provider | Tier | Recovery Time |
|----------|------|--------------|
| S3 Glacier | Expedited | 1–5 minutes |
| S3 Glacier | Standard | 3–5 hours |
| S3 Glacier | Bulk | 5–12 hours |
| S3 Deep Archive | Standard | 12 hours |
| S3 Deep Archive | Bulk | 48 hours |
| GCS Archive | Rewrite | Minutes to hours |
| Azure Archive | High priority | < 1 hour |
| Azure Archive | Standard | 1–15 hours |

---

## Cost Estimation Guidance

Approximate relative pricing (varies by region):

```
STANDARD:     $0.023/GB/month  + $0.0004/1k GET
STANDARD_IA:  $0.0125/GB/month + $0.001/1k GET  (30-day min duration)
GLACIER:      $0.004/GB/month  + retrieval fee  (90-day min duration)
DEEP_ARCHIVE: $0.00099/GB/month + retrieval fee (180-day min duration)
```

Break-even analysis:
- Move to STANDARD_IA if objects are accessed < once per month.
- Move to GLACIER if objects are accessed < once per quarter.
- Move to DEEP_ARCHIVE if objects are accessed < once per year.

## Anti-Patterns

- Do not archive objects with frequent access — retrieval fees exceed storage savings.
- Do not delete minimum-duration objects early — early deletion fees apply (AWS: 30/90/180 days).
- Always test restore procedures — do not discover restore latency during an incident.
- Do not combine auto-tiering (Intelligent-Tiering) with manual lifecycle rules on the same objects — they conflict.
- Separate operational backups (need <1h recovery) from compliance archives (days acceptable) — use different tiers.
