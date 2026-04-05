# Milvus Skill Guide

## Deployment Modes

| Mode | Use Case | Setup |
|------|----------|-------|
| **Lite** | Local dev, single process | `pip install pymilvus[model]`; in-process |
| **Standalone** | Single node, production-ready | Docker Compose |
| **Distributed** | Horizontal scale, >1B vectors | Kubernetes + Helm |

```bash
# Standalone via Docker
wget https://github.com/milvus-io/milvus/releases/download/v2.4.0/milvus-standalone-docker-compose.yml
docker compose -f milvus-standalone-docker-compose.yml up -d
```

```python
from pymilvus import MilvusClient

# Milvus Lite (dev)
client = MilvusClient("milvus_demo.db")

# Standalone / Distributed
client = MilvusClient(
    uri="http://localhost:19530",
    token="root:Milvus",    # default creds; change in production
)
```

## Collection Schema

```python
from pymilvus import MilvusClient, DataType

schema = MilvusClient.create_schema(
    auto_id=False,
    enable_dynamic_field=True,   # allows extra fields not in schema
)

# Primary key field
schema.add_field(field_name="id",        data_type=DataType.INT64,   is_primary=True)
# Dense vector field
schema.add_field(field_name="embedding", data_type=DataType.FLOAT_VECTOR, dim=1536)
# Scalar fields
schema.add_field(field_name="text",      data_type=DataType.VARCHAR,  max_length=4096)
schema.add_field(field_name="category",  data_type=DataType.VARCHAR,  max_length=64)
schema.add_field(field_name="views",     data_type=DataType.INT32)

client.create_collection(
    collection_name="articles",
    schema=schema,
)
```

## Index Configuration

```python
index_params = client.prepare_index_params()

index_params.add_index(
    field_name="embedding",
    index_type="HNSW",           # FLAT | IVF_FLAT | HNSW | DISKANN
    metric_type="COSINE",        # L2 | IP | COSINE
    params={
        "M": 16,                 # HNSW: number of bi-directional links
        "efConstruction": 64,    # HNSW: build-time search width
    },
)

client.create_index("articles", index_params)
client.load_collection("articles")   # load into memory before search
```

### Index Type Comparison

| Type | Accuracy | Speed | Memory | Notes |
|------|----------|-------|--------|-------|
| FLAT | Exact | Slow | High | Best for <1M vectors |
| IVF_FLAT | High | Fast | Medium | nlist=1024 typical |
| HNSW | High | Fastest | High | Best recall/speed trade-off |
| DISKANN | High | Medium | Low | Disk-based; >100M vectors |

## Insert

```python
data = [
    {
        "id": 1,
        "embedding": [0.1, 0.2, ..., 0.9],  # 1536 dims
        "text": "Introduction to vector databases",
        "category": "tutorial",
        "views": 1500,
    },
    # ...
]

result = client.insert(collection_name="articles", data=data)
print(result["insert_count"])
```

## Search

```python
query_vector = get_embedding("vector database performance")

results = client.search(
    collection_name="articles",
    data=[query_vector],
    anns_field="embedding",
    limit=10,
    output_fields=["text", "category", "views"],
    search_params={
        "metric_type": "COSINE",
        "params": {"ef": 100},       # HNSW search-time ef; higher = better recall
    },
)

for hit in results[0]:
    print(hit["id"], hit["distance"], hit["entity"]["text"])
```

## Filtered Search

```python
results = client.search(
    collection_name="articles",
    data=[query_vector],
    anns_field="embedding",
    limit=10,
    filter='category == "tutorial" AND views > 500',
    output_fields=["text", "category"],
    search_params={"metric_type": "COSINE", "params": {"ef": 100}},
)
```

## Query (Non-Vector Filter)

```python
# Fetch by scalar filter without ANN
rows = client.query(
    collection_name="articles",
    filter='views > 10000',
    output_fields=["id", "text", "views"],
    limit=50,
)
```

## Partitions for Data Isolation

```python
# Create partition per tenant or time bucket
client.create_partition("articles", partition_name="2024_Q1")

client.insert(
    collection_name="articles",
    data=data,
    partition_name="2024_Q1",
)

# Search within a specific partition
results = client.search(
    collection_name="articles",
    data=[query_vector],
    anns_field="embedding",
    limit=10,
    partition_names=["2024_Q1"],
    search_params={"metric_type": "COSINE", "params": {"ef": 100}},
)
```

## Dynamic Schema

```python
# enable_dynamic_field=True allows arbitrary extra fields per record
schema = MilvusClient.create_schema(auto_id=False, enable_dynamic_field=True)
schema.add_field("id", DataType.INT64, is_primary=True)
schema.add_field("embedding", DataType.FLOAT_VECTOR, dim=768)

# Insert with extra dynamic fields — no schema change needed
client.insert("docs", [
    {"id": 1, "embedding": [...], "title": "Doc A", "lang": "en", "score": 0.95},
])

# Filter on dynamic fields with $meta prefix not needed — use field name directly
results = client.query("docs", filter='lang == "en"', output_fields=["title", "lang"])
```

## Streaming Insert (Buffer Pattern)

```python
# For high-throughput ingestion, batch inserts in memory before flushing
BATCH_SIZE = 1000
buffer = []

for record in stream:
    buffer.append(record)
    if len(buffer) >= BATCH_SIZE:
        client.insert("articles", buffer)
        buffer.clear()

# Flush remaining
if buffer:
    client.insert("articles", buffer)

# Explicit flush ensures data is searchable (auto-flush interval is 1s by default)
client.flush("articles")
```

## Collection Management

```python
client.release_collection("articles")  # unload from memory (save RAM)
client.drop_collection("articles")

# List collections
print(client.list_collections())

# Collection stats
print(client.get_collection_stats("articles"))
```

## Anti-Patterns

- Do not search before `load_collection()` — raises error.
- Do not use FLAT index for >1M vectors — too slow.
- Always call `flush()` after bulk insert when immediate searchability is required.
- Partition key isolation is more efficient than per-collection multi-tenancy at scale.
