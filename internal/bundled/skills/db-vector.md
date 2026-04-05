# pgvector Skill Guide

## Installation & Setup

```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

```bash
# Install pgvector (Postgres 15+)
apt install postgresql-15-pgvector
# or build from source
```

## Column Types

```sql
CREATE TABLE embeddings (
    id          BIGSERIAL PRIMARY KEY,
    content     TEXT NOT NULL,
    embedding   vector(1536),   -- OpenAI ada-002 dimension
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Supported dimensions: up to 16,000. Common: 1536 (OpenAI), 768 (BERT), 384 (MiniLM), 3072 (OpenAI large).

## Index Types

### IVFFLAT (Inverted File with Flat Compression)

```sql
-- Build after bulk insert; lists ≈ sqrt(row_count)
CREATE INDEX ON embeddings USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

-- Increase probes at query time for higher recall (default 1)
SET ivfflat.probes = 10;
```

### HNSW (Hierarchical Navigable Small World)

```sql
-- Better recall than IVFFLAT; can query immediately after insert
CREATE INDEX ON embeddings USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- Increase ef_search at query time (default 40)
SET hnsw.ef_search = 100;
```

HNSW is preferred for most workloads — no need to rebuild after inserts.

## Similarity Operators

| Operator | Distance | Index op class |
|----------|----------|----------------|
| `<->` | L2 (Euclidean) | `vector_l2_ops` |
| `<=>` | Cosine | `vector_cosine_ops` |
| `<#>` | Inner product (negated) | `vector_ip_ops` |

Use cosine similarity for normalized embeddings (language models).
Use inner product for models already trained with dot-product objectives.

## Similarity Search

```sql
-- Top-10 nearest neighbors by cosine distance
SELECT id, content, embedding <=> $1 AS distance
FROM embeddings
ORDER BY embedding <=> $1
LIMIT 10;

-- With metadata filter (filter before ANN; may hurt recall)
SELECT id, content
FROM embeddings
WHERE category = 'docs'
ORDER BY embedding <=> $1
LIMIT 10;

-- Pre-filter then re-rank pattern for better recall
WITH candidates AS (
    SELECT id, content, embedding
    FROM embeddings
    WHERE category = 'docs'
    LIMIT 200
)
SELECT id, content, embedding <=> $1 AS distance
FROM candidates
ORDER BY distance
LIMIT 10;
```

## Batch Insert with COPY

```sql
-- Fastest bulk load — disable index during load, rebuild after
ALTER INDEX embeddings_embedding_idx RENAME TO embeddings_embedding_idx_old;
DROP INDEX embeddings_embedding_idx_old;

COPY embeddings (content, embedding) FROM STDIN WITH (FORMAT csv);
<content_1>,"[0.1,0.2,...,0.9]"
\.

-- Rebuild index after bulk load
CREATE INDEX ON embeddings USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
```

```python
# Python with psycopg2 + pgvector
import psycopg2
from pgvector.psycopg2 import register_vector
import numpy as np

conn = psycopg2.connect(DATABASE_URL)
register_vector(conn)

cur = conn.cursor()
# Batch insert
rows = [(text, np.array(emb)) for text, emb in pairs]
cur.executemany(
    "INSERT INTO embeddings (content, embedding) VALUES (%s, %s)",
    rows
)
conn.commit()
```

## Distance to Similarity Conversion

```sql
-- Cosine similarity from cosine distance
SELECT 1 - (embedding <=> $1) AS similarity
FROM embeddings
ORDER BY embedding <=> $1
LIMIT 10;
```

## Maintenance

```sql
-- Check index size
SELECT pg_size_pretty(pg_relation_size('embeddings_embedding_idx'));

-- Vacuum after many deletes (HNSW marks nodes deleted, needs vacuum)
VACUUM embeddings;

-- Monitor HNSW graph health
SELECT * FROM pg_stat_user_indexes WHERE indexname LIKE '%embedding%';
```

## Anti-Patterns

- Do not create IVFFLAT index before bulk loading — rebuild after.
- Do not use `<->` (L2) on unnormalized embeddings when cosine is intended.
- Avoid `WHERE` filters before ANN on high-cardinality predicates — use pre-filter + re-rank.
- Dimensions must match exactly; mismatched dimensions raise a cast error.
