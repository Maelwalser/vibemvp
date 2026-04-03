# Azure Blob Storage Skill Guide

## Installation

```bash
pip install azure-storage-blob azure-identity
npm install @azure/storage-blob @azure/identity
dotnet add package Azure.Storage.Blobs Azure.Identity
```

## Authentication (Recommended: DefaultAzureCredential)

```python
from azure.storage.blob import BlobServiceClient
from azure.identity import DefaultAzureCredential

credential = DefaultAzureCredential()
service_client = BlobServiceClient(
    account_url=f"https://{os.environ['AZURE_STORAGE_ACCOUNT']}.blob.core.windows.net",
    credential=credential,
)
```

```typescript
import { BlobServiceClient } from "@azure/storage-blob";
import { DefaultAzureCredential } from "@azure/identity";

const credential = new DefaultAzureCredential();
const serviceClient = new BlobServiceClient(
  `https://${process.env.AZURE_STORAGE_ACCOUNT}.blob.core.windows.net`,
  credential
);
```

For local development: use connection string.

```python
service_client = BlobServiceClient.from_connection_string(os.environ["AZURE_STORAGE_CONNECTION_STRING"])
```

## Container Naming Conventions

Azure container names must be 3–63 characters, lowercase letters, digits, and hyphens only.

Recommended naming patterns:
```
{purpose}-{env}           → uploads-prod, backups-staging
{tenant}-{purpose}        → acme-documents, globex-assets
{retention}-{purpose}     → 30d-logs, 7y-invoices
```

```python
container_client = service_client.get_container_client("uploads-prod")

# Create if not exists
container_client.create_container(exist_ok=True)
```

## Upload / Download / Delete

```python
blob_client = service_client.get_blob_client(container="uploads-prod", blob="documents/report.pdf")

# Upload
with open("/tmp/report.pdf", "rb") as f:
    blob_client.upload_blob(
        f,
        content_settings=ContentSettings(content_type="application/pdf"),
        overwrite=True,
    )

# Upload with metadata and tags
blob_client.upload_blob(
    data,
    metadata={"uploaded_by": "user123", "source": "web-ui"},
    tags={"department": "finance", "retention": "7y"},
    overwrite=True,
)

# Download
download = blob_client.download_blob()
data = download.readall()

# Download to file
with open("/tmp/downloaded.pdf", "wb") as f:
    download_stream = blob_client.download_blob()
    download_stream.readinto(f)

# Delete
blob_client.delete_blob()
```

## User Delegation SAS URI Generation

User Delegation SAS is more secure than account key SAS — uses Azure AD identity, not the storage account key.

```python
from azure.storage.blob import (
    BlobServiceClient,
    BlobSasPermissions,
    generate_blob_sas,
    UserDelegationKey,
)
from azure.identity import DefaultAzureCredential
from datetime import datetime, timezone, timedelta

credential = DefaultAzureCredential()
service_client = BlobServiceClient(
    account_url=f"https://{account_name}.blob.core.windows.net",
    credential=credential,
)

# Get user delegation key (valid up to 7 days)
delegation_key: UserDelegationKey = service_client.get_user_delegation_key(
    key_start_time=datetime.now(timezone.utc),
    key_expiry_time=datetime.now(timezone.utc) + timedelta(days=1),
)

# Generate SAS token
sas_token = generate_blob_sas(
    account_name=account_name,
    container_name="uploads-prod",
    blob_name="documents/report.pdf",
    user_delegation_key=delegation_key,
    permission=BlobSasPermissions(read=True),
    expiry=datetime.now(timezone.utc) + timedelta(minutes=15),
)

sas_url = f"https://{account_name}.blob.core.windows.net/uploads-prod/documents/report.pdf?{sas_token}"
```

```typescript
import { generateBlobSASQueryParameters, BlobSASPermissions, SASProtocol } from "@azure/storage-blob";

// Get user delegation key
const userDelegationKey = await serviceClient.getUserDelegationKey(
  new Date(),
  new Date(Date.now() + 24 * 60 * 60 * 1000) // 1 day
);

const sasQuery = generateBlobSASQueryParameters(
  {
    containerName: "uploads-prod",
    blobName: "documents/report.pdf",
    permissions: BlobSASPermissions.parse("r"), // read
    startsOn: new Date(),
    expiresOn: new Date(Date.now() + 15 * 60 * 1000), // 15 min
    protocol: SASProtocol.Https,
  },
  userDelegationKey,
  accountName
);

const sasUrl = `https://${accountName}.blob.core.windows.net/uploads-prod/documents/report.pdf?${sasQuery}`;
```

## Lifecycle Policy (Hot → Cool → Cold → Archive)

```python
from azure.mgmt.storage import StorageManagementClient
from azure.mgmt.storage.models import (
    ManagementPolicy, ManagementPolicyRule, ManagementPolicyDefinition,
    ManagementPolicyAction, ManagementPolicyBaseBlob,
    ManagementPolicyFilter, ManagementPolicySnapShot,
)

# Via Azure SDK for management operations
mgmt_client = StorageManagementClient(credential, subscription_id)

policy = ManagementPolicy(
    policy={
        "rules": [
            {
                "name": "tiering-rule",
                "enabled": True,
                "type": "Lifecycle",
                "definition": {
                    "filters": {
                        "blobTypes": ["blockBlob"],
                        "prefixMatch": ["uploads/"],
                    },
                    "actions": {
                        "baseBlob": {
                            "tierToCool": {"daysAfterModificationGreaterThan": 30},
                            "tierToCold": {"daysAfterModificationGreaterThan": 60},
                            "tierToArchive": {"daysAfterModificationGreaterThan": 180},
                            "delete": {"daysAfterModificationGreaterThan": 365},
                        }
                    },
                },
            }
        ]
    }
)

mgmt_client.management_policies.create_or_update(
    resource_group_name, account_name, "default", policy
)
```

JSON policy (via Azure Portal or ARM template):

```json
{
  "rules": [
    {
      "name": "tiering-rule",
      "enabled": true,
      "type": "Lifecycle",
      "definition": {
        "filters": { "blobTypes": ["blockBlob"], "prefixMatch": ["uploads/"] },
        "actions": {
          "baseBlob": {
            "tierToCool":    { "daysAfterModificationGreaterThan": 30  },
            "tierToCold":    { "daysAfterModificationGreaterThan": 60  },
            "tierToArchive": { "daysAfterModificationGreaterThan": 180 },
            "delete":        { "daysAfterModificationGreaterThan": 365 }
          }
        }
      }
    }
  ]
}
```

## Blob Metadata and Tags

```python
# Metadata: key-value pairs, returned in blob properties
blob_client.set_blob_metadata({"project": "apollo", "version": "2"})

# Tags: key-value pairs, queryable across the entire storage account
blob_client.set_blob_tags({"department": "engineering", "cost-center": "1234"})

# Find blobs by tag (across containers)
results = service_client.find_blobs_by_tags('department = "engineering"')
for blob in results:
    print(blob.name, blob.container_name)
```

## Redundancy Options

| Option | Abbreviation | Copies | Recovery |
|--------|-------------|--------|---------|
| Locally Redundant | LRS | 3 (same datacenter) | Hardware failure |
| Zone Redundant | ZRS | 3 (3 availability zones) | Zone outage |
| Geo Redundant | GRS | 6 (3 primary + 3 secondary region) | Region disaster |
| Geo Zone Redundant | GZRS | 6 (ZRS primary + 3 secondary) | Zone + region |

Choose based on RTO/RPO requirements:
- **LRS**: Dev/test, non-critical
- **ZRS**: Production OLTP, high availability within region
- **GRS/GZRS**: Compliance, cross-region disaster recovery

## Encryption at Rest (Customer-Managed Keys)

```bash
# Enable customer-managed keys via Azure Key Vault
az storage account update \
  --name mystorageaccount \
  --resource-group mygroup \
  --encryption-key-source Microsoft.Keyvault \
  --encryption-key-vault https://mykeyvault.vault.azure.net/ \
  --encryption-key-name mykey \
  --encryption-key-version <key-version>
```

## List Blobs

```python
container_client = service_client.get_container_client("uploads-prod")

# List with prefix filter
blobs = container_client.list_blobs(name_starts_with="documents/")
for blob in blobs:
    print(blob.name, blob.size, blob.last_modified)

# Paginated listing (for large containers)
pages = container_client.list_blobs().by_page(results_per_page=100)
for page in pages:
    for blob in page:
        process(blob)
```

## Anti-Patterns

- Never use storage account connection strings in production code — use `DefaultAzureCredential` with managed identity.
- Do not use account key SAS in production — prefer user delegation SAS (tied to Azure AD identity, revocable).
- Do not skip lifecycle policies for user-uploaded content — unmanaged blobs accumulate cost.
- Archive tier blobs must be rehydrated before reading — plan 1–15 hours for rehydration.
- Container names cannot contain uppercase letters — convert to lowercase before creating.
