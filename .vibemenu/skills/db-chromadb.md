# Chroma Skill Guide

## Installation

```bash
pip install chromadb
# With embedding model support
pip install chromadb sentence-transformers
```

## Client Setup

```python
import chromadb

# Persistent (survives restart) — recommended for production
client = chromadb.PersistentClient(path="/data/chroma")

# Ephemeral (in-memory) — for testing/dev
client = chromadb.EphemeralClient()

# HTTP client (connect to remote Chroma server)
client = chromadb.HttpClient(host="localhost", port=8000)
```

### Run Chroma Server

```bash
chroma run --path /data/chroma --port 8000
```

## Collections

```python
# Get or create
collection = client.get_or_create_collection(
    name="documents",
    metadata={"hnsw:space": "cosine"},   # "l2" | "ip" | "cosine"
)

# List all collections
print(client.list_collections())

# Delete
client.delete_collection("documents")
```

## Adding Documents

```python
# Chroma auto-embeds if you supply documents (uses default embedding function)
collection.add(
    documents=["The quick brown fox", "Vector databases are fast"],
    metadatas=[{"source": "wiki"}, {"source": "blog"}],
    ids=["doc_1", "doc_2"],
)

# Supply pre-computed embeddings (skip auto-embedding)
collection.add(
    embeddings=[[0.1, 0.2, ...], [0.3, 0.4, ...]],
    documents=["Raw text for reference"],
    metadatas=[{"source": "custom"}],
    ids=["emb_1"],
)
```

IDs must be unique strings. Duplicate IDs update the existing record.

## Querying

```python
# Query by text (auto-embeds the query)
results = collection.query(
    query_texts=["fast similarity search"],
    n_results=5,
    include=["documents", "metadatas", "distances"],
)

# Query by pre-computed vector
results = collection.query(
    query_embeddings=[[0.1, 0.2, ...]],
    n_results=10,
)

# Access results
for doc, meta, dist in zip(
    results["documents"][0],
    results["metadatas"][0],
    results["distances"][0],
):
    print(f"{dist:.4f} | {meta['source']} | {doc[:80]}")
```

## Custom Embedding Function

```python
from chromadb import EmbeddingFunction, Embeddings
from sentence_transformers import SentenceTransformer

class SentenceTransformerEF(EmbeddingFunction):
    def __init__(self, model_name: str = "all-MiniLM-L6-v2"):
        self.model = SentenceTransformer(model_name)

    def __call__(self, input: list[str]) -> Embeddings:
        return self.model.encode(input).tolist()

# Use custom embedding function
ef = SentenceTransformerEF("all-MiniLM-L6-v2")
collection = client.get_or_create_collection(
    name="custom_docs",
    embedding_function=ef,
    metadata={"hnsw:space": "cosine"},
)

collection.add(documents=["Hello world"], ids=["d1"])
results = collection.query(query_texts=["greeting"], n_results=3)
```

## Metadata Filtering

```python
# Equality filter
results = collection.query(
    query_texts=["machine learning"],
    n_results=10,
    where={"source": "arxiv"},
)

# Logical operators
results = collection.query(
    query_texts=["deep learning"],
    n_results=10,
    where={
        "$and": [
            {"source": {"$in": ["arxiv", "blog"]}},
            {"year": {"$gte": 2022}},
        ]
    },
)

# Supported operators: $eq $ne $gt $gte $lt $lte $in $nin $and $or
```

## Document Chunking Strategy

```python
def chunk_text(text: str, chunk_size: int = 512, overlap: int = 64) -> list[str]:
    """Split text into overlapping chunks for better retrieval."""
    words = text.split()
    chunks = []
    start = 0
    while start < len(words):
        end = min(start + chunk_size, len(words))
        chunks.append(" ".join(words[start:end]))
        start += chunk_size - overlap
    return chunks

def add_document(collection, doc_id: str, text: str, metadata: dict):
    chunks = chunk_text(text)
    collection.add(
        documents=chunks,
        ids=[f"{doc_id}_chunk_{i}" for i in range(len(chunks))],
        metadatas=[{**metadata, "chunk_index": i} for i in range(len(chunks))],
    )
```

## RAG Pipeline Integration

```python
from anthropic import Anthropic

anthropic = Anthropic()

def rag_query(user_question: str, collection, n_results: int = 5) -> str:
    # 1. Retrieve relevant chunks
    results = collection.query(
        query_texts=[user_question],
        n_results=n_results,
        include=["documents", "metadatas"],
    )
    
    context_chunks = results["documents"][0]
    context = "\n\n---\n\n".join(context_chunks)
    
    # 2. Generate answer with context
    response = anthropic.messages.create(
        model="claude-opus-4-6",
        max_tokens=1024,
        messages=[{
            "role": "user",
            "content": f"Context:\n{context}\n\nQuestion: {user_question}",
        }],
        system="Answer the question using only the provided context. If the answer is not in the context, say so.",
    )
    return response.content[0].text
```

## Update & Delete

```python
# Update existing documents (must exist)
collection.update(
    ids=["doc_1"],
    documents=["Updated content"],
    metadatas=[{"source": "wiki", "version": 2}],
)

# Upsert (insert or update)
collection.upsert(
    documents=["New or updated content"],
    ids=["doc_1"],
    metadatas=[{"source": "wiki"}],
)

# Delete by IDs
collection.delete(ids=["doc_1", "doc_2"])

# Delete by filter
collection.delete(where={"source": "spam"})
```

## Collection Stats

```python
print(collection.count())   # number of documents
print(collection.peek())    # first 10 records for inspection
```

## Anti-Patterns

- Do not use EphemeralClient in production — data is lost on restart.
- Do not use chunk_size > 1024 tokens — retrieval quality degrades.
- Avoid storing embeddings without documents — you lose the source text for RAG.
- Do not skip overlap when chunking — boundary sentences lose context.
- IDs must be strings; integer IDs cause silent errors.
