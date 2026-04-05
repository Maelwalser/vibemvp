# Weaviate Skill Guide

## Installation

```bash
# Docker Compose (development)
docker run -d \
  -e QUERY_DEFAULTS_LIMIT=25 \
  -e AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED=true \
  -e PERSISTENCE_DATA_PATH=/var/lib/weaviate \
  -p 8080:8080 \
  semitechnologies/weaviate:latest

pip install weaviate-client
```

```python
import weaviate

client = weaviate.connect_to_local()          # localhost:8080
# or
client = weaviate.connect_to_weaviate_cloud(
    cluster_url=os.environ["WCD_URL"],
    auth_credentials=weaviate.auth.AuthApiKey(os.environ["WCD_API_KEY"]),
)
```

## Class Schema Definition

```python
from weaviate.classes.config import Configure, Property, DataType

client.collections.create(
    name="Article",
    vectorizer_config=Configure.Vectorizer.text2vec_openai(
        model="text-embedding-3-small",
        dimensions=1536,
    ),
    # Or use Cohere:
    # vectorizer_config=Configure.Vectorizer.text2vec_cohere(model="embed-multilingual-v3.0"),
    generative_config=Configure.Generative.openai(model="gpt-4o"),
    properties=[
        Property(name="title",   data_type=DataType.TEXT),
        Property(name="body",    data_type=DataType.TEXT),
        Property(name="author",  data_type=DataType.TEXT),
        Property(name="pubDate", data_type=DataType.DATE),
        Property(name="views",   data_type=DataType.INT),
    ],
)
```

## Semantic Search (near_text)

```python
articles = client.collections.get("Article")

response = articles.query.near_text(
    query="machine learning breakthroughs",
    limit=5,
    return_metadata=weaviate.classes.query.MetadataQuery(distance=True),
)

for obj in response.objects:
    print(obj.properties["title"], obj.metadata.distance)
```

## Vector Search (near_vector)

```python
# When you already have an embedding vector
my_vector = get_embedding("some query text")

response = articles.query.near_vector(
    near_vector=my_vector,
    limit=10,
    distance=0.3,   # max distance threshold (optional)
)
```

## Hybrid Search

```python
# alpha: 0.0 = pure BM25 keyword, 1.0 = pure vector
response = articles.query.hybrid(
    query="quantum computing applications",
    alpha=0.7,          # 70% vector, 30% keyword
    limit=10,
    return_metadata=weaviate.classes.query.MetadataQuery(score=True, explain_score=True),
)

for obj in response.objects:
    print(obj.properties["title"], obj.metadata.score)
```

## Metadata Filtering (where)

```python
from weaviate.classes.query import Filter

response = articles.query.near_text(
    query="climate change",
    limit=10,
    filters=Filter.by_property("views").greater_than(1000),
)

# Compound filter
response = articles.query.hybrid(
    query="renewable energy",
    alpha=0.5,
    limit=10,
    filters=(
        Filter.by_property("views").greater_than(500) &
        Filter.by_property("author").equal("Jane Doe")
    ),
)
```

## Cross-References Between Classes

```python
# Define reference property in schema
client.collections.create(
    name="Comment",
    properties=[
        Property(name="text", data_type=DataType.TEXT),
        Property(
            name="hasArticle",
            data_type=DataType.OBJECT,
            nested_properties=[],
        ),
    ],
    references=[
        weaviate.classes.config.ReferenceProperty(
            name="hasArticle",
            target_collection="Article",
        )
    ],
)

# Insert with cross-reference
import uuid
article_id = uuid.uuid4()
comment_id = comments.data.insert(
    properties={"text": "Great article!"},
    references={"hasArticle": article_id},
)

# Query with cross-reference
from weaviate.classes.query import QueryReference

response = comments.query.fetch_objects(
    return_references=QueryReference(link_on="hasArticle", return_properties=["title"]),
    limit=5,
)
```

## Batch Import

```python
# Recommended for bulk loading (>100 objects)
articles = client.collections.get("Article")

with articles.batch.dynamic() as batch:
    for row in dataset:
        batch.add_object(
            properties={
                "title":   row["title"],
                "body":    row["body"],
                "author":  row["author"],
                "pubDate": row["pub_date"],
                "views":   row["views"],
            },
            # Optionally supply your own vector (skips vectorizer)
            # vector=row["precomputed_embedding"],
        )

# Check for errors after batch
if articles.batch.failed_objects:
    print(f"Failed: {len(articles.batch.failed_objects)}")
```

## Generative Search (RAG)

```python
from weaviate.classes.query import GenerativeQuery

response = articles.generate.near_text(
    query="quantum computing",
    limit=3,
    grouped_task="Summarize these articles in 3 bullet points.",
    # single_prompt="Explain this article in one sentence: {title} - {body}",
)

print(response.generated)
```

## Delete Collection / Objects

```python
client.collections.delete("Article")   # drop entire collection

articles.data.delete_by_id(object_id)  # single object

# Batch delete by filter
articles.data.delete_many(
    where=Filter.by_property("author").equal("Spam Author")
)
```

## Anti-Patterns

- Do not mix `near_text` and `near_vector` in the same query — use one.
- HNSW index is built incrementally; no need to rebuild after inserts.
- `alpha=0.5` hybrid is a safe default when unsure; tune based on eval metrics.
- Always set a `distance` or `certainty` threshold to avoid irrelevant results at tail.
