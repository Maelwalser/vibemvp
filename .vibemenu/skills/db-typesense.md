# Typesense Skill Guide

## Overview

Typesense is an open-source, typo-tolerant search engine built for speed with instant search, faceting, geo-search, and vector search. It is designed to be simpler to operate than Elasticsearch.

## Setup & Connection

```javascript
import Typesense from 'typesense';

const client = new Typesense.Client({
  nodes: [{ host: process.env.TYPESENSE_HOST, port: 443, protocol: 'https' }],
  apiKey: process.env.TYPESENSE_API_KEY,
  connectionTimeoutSeconds: 5,
  retryIntervalSeconds: 0.1,
  healthcheckIntervalSeconds: 60,
});
```

```bash
# Docker (development)
docker run -d --name typesense \
  -p 8108:8108 \
  -v $(pwd)/typesense-data:/data \
  typesense/typesense:27.0 \
  --data-dir /data \
  --api-key=xyz \
  --enable-cors
```

## Collection Schema

```javascript
// Create collection with typed schema
await client.collections().create({
  name: 'products',
  fields: [
    { name: 'id',           type: 'string' },
    { name: 'name',         type: 'string',  facet: false },
    { name: 'description',  type: 'string',  optional: true },
    { name: 'brand',        type: 'string',  facet: true },       // facet: true = filterable + facet count
    { name: 'category',     type: 'string',  facet: true },
    { name: 'tags',         type: 'string[]', facet: true },
    { name: 'price',        type: 'float',   facet: true },
    { name: 'rating',       type: 'float',   optional: true },
    { name: 'stock',        type: 'int32' },
    { name: 'active',       type: 'bool',    facet: true },
    { name: 'location',     type: 'geopoint', optional: true },   // [lat, lng]
    { name: 'embedding',    type: 'float[]', num_dim: 1536,        // vector field
      optional: true, index: true },
    { name: 'createdAt',    type: 'int64' },                      // Unix timestamp
  ],
  default_sorting_field: 'rating',  // default sort when no sort specified
  enable_nested_fields: true,       // allow dot-notation for nested objects
});
```

## Indexing Documents

```javascript
// Upsert single document
await client.collections('products').documents().upsert({
  id: 'prod-1',
  name: 'MacBook Pro 14"',
  brand: 'Apple',
  category: 'laptops',
  tags: ['portable', 'professional', 'retina'],
  price: 1999.00,
  rating: 4.8,
  stock: 15,
  active: true,
  location: [37.7749, -122.4194],  // lat, lng
  createdAt: Math.floor(Date.now() / 1000),
});

// Bulk import (fast — JSONL format)
const documents = products.map(p => JSON.stringify(p)).join('\n');
await client.collections('products').documents().importJsonl(documents, {
  action: 'upsert',
  batch_size: 1000,
});
```

## Search

```javascript
const results = await client.collections('products').documents().search({
  q: 'wireless headphones',
  query_by: 'name,description,tags',       // fields to search (order = weight)
  query_by_weights: '10,5,3',              // relative weight per field
  filter_by: 'active:=true && price:[50..500] && category:=electronics',
  sort_by: '_text_match:desc,rating:desc', // sort by relevance then rating
  facet_by: 'brand,category,tags',
  max_facet_values: 20,
  per_page: 20,
  page: 1,
  highlight_full_fields: 'name',
  snippet_threshold: 30,
  include_fields: 'id,name,price,brand,category,rating',
  num_typos: 2,                            // allow up to 2 typos
  prefix: true,                            // prefix matching (for autocomplete)
  drop_tokens_threshold: 3,               // if no results, drop uncommon tokens
});

console.log(results.hits);        // matched documents with highlights
console.log(results.facet_counts); // { brand: [{ value: 'Apple', count: 5 }], ... }
console.log(results.found);        // total count
```

## Typo Tolerance Configuration

```javascript
// Per-search typo configuration
await client.collections('products').documents().search({
  q: 'headphons',
  query_by: 'name',
  num_typos: 2,              // max typos allowed (0, 1, or 2)
  min_len_1typo: 4,          // min word length to tolerate 1 typo (default: 4)
  min_len_2typo: 7,          // min word length to tolerate 2 typos (default: 7)
  typo_tokens_threshold: 1,  // if ≥1 result without typos, disable typo matching
  split_join_tokens: 'fallback',  // try splitting/joining tokens on no results
});

// Disable typo tolerance for specific fields (e.g., brand names)
// Set in schema: { name: 'brand', type: 'string', facet: true, locale: '' }
```

## Faceted Search

```javascript
const results = await client.collections('products').documents().search({
  q: 'laptop',
  query_by: 'name,description',
  filter_by: 'active:=true',
  facet_by: 'brand,category,price(0, 500, 1000, 2000, 5000)',  // price range facets
  max_facet_values: 10,
  facet_query: 'brand:app',   // filter facet values matching "app" (for facet search)
  per_page: 0,                // return only facets, no documents
});

// Parse facet results
results.facet_counts.forEach(facet => {
  console.log(`${facet.field_name}:`);
  facet.counts.forEach(c => console.log(`  ${c.value}: ${c.count}`));
});
```

## Geo Search

```javascript
// Find products near a location (sorted by distance)
const results = await client.collections('stores').documents().search({
  q: '*',
  query_by: 'name',
  filter_by: 'location:(37.7749, -122.4194, 10 km)',  // within 10km of SF
  sort_by: 'location(37.7749, -122.4194):asc',         // closest first
  per_page: 20,
});

// Bounding box
filter_by: 'location:(37.80, -122.45, 37.70, -122.38)'  // NE lat/lng, SW lat/lng
```

## Vector Search

```javascript
// Search by vector embedding
const results = await client.collections('products').documents().search({
  q: '*',
  vector_query: `embedding:([${queryEmbedding.join(',')}], k:20)`,  // top 20 by vector
  filter_by: 'active:=true',
  per_page: 20,
});

// Hybrid search: vector + keyword
const results = await client.collections('products').documents().search({
  q: 'comfortable wireless audio',
  query_by: 'name,description',
  vector_query: `embedding:([${queryEmbedding.join(',')}], k:100, distance_threshold:0.3)`,
  per_page: 20,
});
```

## Multi-Search for Federated Results

```javascript
const { results } = await client.multiSearch.perform({
  searches: [
    { collection: 'products', q: 'apple',  query_by: 'name,brand', per_page: 5 },
    { collection: 'articles', q: 'apple',  query_by: 'title,body', per_page: 5 },
    { collection: 'brands',   q: 'apple',  query_by: 'name',       per_page: 5 },
  ],
}, { query_by: 'name' });  // default params applied to all searches
```

## Search-Only API Key (Frontend Safe)

```javascript
// Generate scoped key for frontend — read-only, filtered to user's data
const scopedKey = client.keys().generateScopedSearchKey(
  process.env.TYPESENSE_SEARCH_KEY,  // parent search key
  {
    filter_by: `userId:=${currentUserId}`,   // enforce data isolation
    expires_at: Math.floor(Date.now() / 1000) + 3600,  // 1 hour
  }
);
```

## Key Rules

- `facet: true` must be set at schema creation — cannot be added later without re-indexing
- `num_dim` on vector fields must match the embedding model's output dimension exactly
- Use `action: 'upsert'` for bulk import to handle duplicate IDs gracefully
- Typesense stores data in memory + disk — ensure RAM ≥ collection size for fast search
- `sort_by: '_text_match:desc'` uses BM25-like relevance score; combine with a tiebreaker field
- `query_by` field order matters for relevance — list most important fields first
- Typesense does not support full aggregations like Elasticsearch — use `facet_by` for counts
- For very large datasets (>50M docs), Typesense Cloud or a cluster deployment is recommended
