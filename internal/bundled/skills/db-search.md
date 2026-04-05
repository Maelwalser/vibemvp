# Elasticsearch & OpenSearch Skill Guide

## Overview

Elasticsearch is a distributed search and analytics engine based on Apache Lucene. OpenSearch is the AWS-maintained fork, API-compatible with Elasticsearch 7.10.2. Both support full-text search, structured queries, aggregations, and vector search.

## Setup & Connection

```javascript
// Node.js — @elastic/elasticsearch
import { Client } from '@elastic/elasticsearch';

const client = new Client({
  node: process.env.ELASTICSEARCH_URL,
  auth: { apiKey: process.env.ELASTICSEARCH_API_KEY },
  tls: { rejectUnauthorized: process.env.NODE_ENV === 'production' },
});

// OpenSearch — @opensearch-project/opensearch
import { Client } from '@opensearch-project/opensearch';
const client = new Client({ node: process.env.OPENSEARCH_URL });
```

## Index Mapping with Analyzers

```javascript
await client.indices.create({
  index: 'products',
  body: {
    settings: {
      number_of_shards: 2,
      number_of_replicas: 1,
      analysis: {
        analyzer: {
          edge_ngram_analyzer: {
            type: 'custom',
            tokenizer: 'standard',
            filter: ['lowercase', 'edge_ngram_filter'],
          },
          autocomplete_search: {
            type: 'custom',
            tokenizer: 'standard',
            filter: ['lowercase'],
          },
        },
        filter: {
          edge_ngram_filter: {
            type: 'edge_ngram',
            min_gram: 2,
            max_gram: 20,
          },
        },
      },
    },
    mappings: {
      properties: {
        name: {
          type: 'text',
          analyzer: 'edge_ngram_analyzer',   // index with edge ngrams for autocomplete
          search_analyzer: 'autocomplete_search',
          fields: {
            keyword: { type: 'keyword' },    // for exact match / aggregation
            standard: { type: 'text', analyzer: 'standard' }, // for full-text
          },
        },
        description: { type: 'text', analyzer: 'standard' },
        price:       { type: 'double' },
        category:    { type: 'keyword' },    // exact match, aggregation
        tags:        { type: 'keyword' },
        stock:       { type: 'integer' },
        active:      { type: 'boolean' },
        createdAt:   { type: 'date' },
        // Vector search field
        embedding:   { type: 'dense_vector', dims: 1536, index: true, similarity: 'cosine' },
      },
    },
  },
});
```

## Bool Query (must / should / filter / must_not)

```javascript
const results = await client.search({
  index: 'products',
  body: {
    query: {
      bool: {
        must: [
          { match: { name: { query: 'wireless headphones', operator: 'and' } } },
        ],
        should: [
          { match: { description: 'noise cancelling' } },
          { term: { tags: 'premium' } },
        ],
        filter: [    // filter does NOT affect relevance score (faster)
          { term:  { active: true } },
          { term:  { category: 'electronics' } },
          { range: { price: { gte: 50, lte: 500 } } },
        ],
        must_not: [
          { term: { tags: 'discontinued' } },
        ],
        minimum_should_match: 1,
      },
    },
    sort: [
      { _score: { order: 'desc' } },
      { price:  { order: 'asc' } },
    ],
    from: 0,
    size: 20,
    highlight: {
      fields: {
        name:        { fragment_size: 150 },
        description: { fragment_size: 200, number_of_fragments: 3 },
      },
    },
  },
});
```

## Aggregations

```javascript
const analytics = await client.search({
  index: 'orders',
  body: {
    size: 0,  // no document hits — aggregations only
    aggs: {
      // Terms aggregation (group by category)
      by_category: {
        terms: { field: 'category', size: 10 },
        aggs: {
          total_revenue: { sum: { field: 'amount' } },
          avg_order:     { avg: { field: 'amount' } },
        },
      },
      // Date histogram aggregation
      orders_over_time: {
        date_histogram: {
          field: 'createdAt',
          calendar_interval: 'day',
          format: 'yyyy-MM-dd',
        },
        aggs: {
          daily_revenue: { sum: { field: 'amount' } },
        },
      },
      // Stats aggregation
      price_stats: {
        stats: { field: 'amount' }, // count, min, max, avg, sum
      },
      // Nested terms
      top_users: {
        terms: { field: 'userId', size: 5, order: { total_spend: 'desc' } },
        aggs: { total_spend: { sum: { field: 'amount' } } },
      },
    },
  },
});
```

## Index Templates

```javascript
// Create an index template (applies to all matching future indexes)
await client.indices.putIndexTemplate({
  name: 'logs-template',
  body: {
    index_patterns: ['logs-*'],
    priority: 100,
    template: {
      settings: {
        number_of_shards: 1,
        number_of_replicas: 1,
        'index.lifecycle.name': 'logs-ilm-policy',
      },
      mappings: {
        properties: {
          '@timestamp': { type: 'date' },
          level:        { type: 'keyword' },
          message:      { type: 'text', analyzer: 'standard' },
          service:      { type: 'keyword' },
          traceId:      { type: 'keyword', index: false },
        },
      },
    },
  },
});
```

## Vector Search (kNN)

```javascript
// Store document with embedding
await client.index({
  index: 'products',
  body: {
    name: 'Wireless Headphones',
    description: '...',
    embedding: embeddingVector,  // Float32Array or number[] of dims: 1536
  },
});

// kNN search — find semantically similar products
const results = await client.search({
  index: 'products',
  body: {
    knn: {
      field: 'embedding',
      query_vector: queryEmbedding,
      k: 10,                   // return top 10 nearest neighbors
      num_candidates: 100,     // candidates to consider per shard
      filter: { term: { active: true } },
    },
    // Hybrid: combine kNN with BM25 full-text
    query: {
      bool: {
        should: [
          { match: { name: userQuery } },
        ],
      },
    },
  },
});
```

## OpenSearch API Compatibility

```javascript
// OpenSearch is compatible with Elasticsearch 7.10.2 API
// Switch client — no query/aggregation changes needed for standard operations

// OpenSearch divergences:
// - Security plugin (fine-grained access control) is built-in (OSS)
// - Some ML/LLM features differ (OpenSearch ML Commons vs Elastic ML)
// - kNN plugin: use "knn_vector" type instead of "dense_vector"

// OpenSearch kNN field mapping
embedding: {
  type: 'knn_vector',
  dimension: 1536,
  method: { name: 'hnsw', space_type: 'cosinesimil', engine: 'nmslib' },
}
```

## RBAC and Field-Level Security

```javascript
// Elasticsearch — define role with field-level security
await client.security.putRole({
  name: 'analyst_role',
  body: {
    indices: [{
      names: ['orders'],
      privileges: ['read'],
      field_security: {
        grant: ['orderId', 'amount', 'createdAt', 'category'],  // only these fields visible
        except: ['userId', 'email', 'address'],                  // these fields hidden
      },
      query: '{"term": {"active": true}}',  // document-level security (only active docs)
    }],
  },
});
```

## Key Rules

- Use `filter` context (not `must`) for non-scoring filters — filters are cached and faster
- Always set `number_of_replicas: 0` during bulk reindexing, then restore after
- Do not use `_id` as a search field — it is not analyzed; use a dedicated `id` keyword field
- Use `refresh=wait_for` (not `refresh=true`) after writes in tests to avoid forcing segment merges
- Avoid wildcard queries on the left side of a pattern (`*word`) — they cannot use indexes
- `terms` aggregation default size is 10 — increase it when you need more buckets
- Use ILM (Index Lifecycle Management) for log indexes: hot → warm → cold → delete
- `dense_vector` with `index: true` enables HNSW ANN indexing (ES 8+); without it, brute-force only
