# Meilisearch Skill Guide

## Overview

Meilisearch is an open-source search engine focused on developer experience with instant search, typo tolerance, and hybrid semantic + keyword search out of the box.

## Setup & Connection

```javascript
import { MeiliSearch } from 'meilisearch';

const client = new MeiliSearch({
  host: process.env.MEILISEARCH_HOST,   // e.g., http://localhost:7700
  apiKey: process.env.MEILISEARCH_API_KEY,
});
```

```bash
# Docker
docker run -d --name meilisearch \
  -p 7700:7700 \
  -e MEILI_MASTER_KEY=your-master-key \
  -v $(pwd)/meili_data:/meili_data \
  getmeili/meilisearch:latest
```

## Index Creation and Configuration

```javascript
// Create index (async — returns task ID)
const task = await client.createIndex('products', { primaryKey: 'id' });
await client.waitForTask(task.taskUid);

const index = client.index('products');

// Configure index settings in one call
await index.updateSettings({
  // Which fields are searched
  searchableAttributes: ['name', 'description', 'brand', 'tags'],

  // Which fields can be used for filtering and faceting
  filterableAttributes: ['category', 'brand', 'price', 'active', 'tags'],

  // Which fields can be used for sorting
  sortableAttributes: ['price', 'createdAt', 'rating'],

  // Ranking rules (order matters — evaluated top to bottom)
  rankingRules: [
    'words',       // documents matching more query words rank higher
    'typo',        // fewer typos = higher rank
    'proximity',   // closer match words = higher rank
    'attribute',   // searchableAttributes order matters
    'sort',        // custom sort criteria
    'exactness',   // exact matches rank higher
  ],

  // Typo tolerance
  typoTolerance: {
    enabled: true,
    minWordSizeForTypos: { oneTypo: 5, twoTypos: 9 },
    disableOnWords: ['iphone', 'macbook'],  // exact brand names
    disableOnAttributes: ['brand'],
  },

  // Fields to exclude from search result (_source)
  displayedAttributes: ['id', 'name', 'description', 'price', 'category', 'rating'],
});

await client.waitForTask((await index.updateSettings({})).taskUid);
```

## Indexing Documents

```javascript
// Add or replace documents (upsert by primaryKey)
const task = await index.addDocuments([
  { id: '1', name: 'MacBook Pro', brand: 'Apple', category: 'laptops', price: 1999, active: true },
  { id: '2', name: 'ThinkPad X1', brand: 'Lenovo', category: 'laptops', price: 1399, active: true },
]);
await client.waitForTask(task.taskUid);

// Partial update (only specified fields)
await index.updateDocuments([
  { id: '1', price: 1799 },   // only price updated
]);

// Delete
await index.deleteDocument('1');
await index.deleteDocuments({ filter: 'active = false' });
```

## Search API

```javascript
// Basic search
const results = await index.search('wireless headphones', {
  limit: 20,
  offset: 0,
});

// Advanced search with filters and facets
const results = await index.search('laptop', {
  filter: [
    'category = laptops',
    'price >= 500 AND price <= 2000',
    'active = true',
    ['brand = Apple', 'brand = Lenovo'],  // OR within inner array
  ],
  facets: ['brand', 'category'],  // return facet counts
  sort: ['price:asc'],            // sort ascending by price
  limit: 20,
  offset: 0,
  attributesToHighlight: ['name', 'description'],
  highlightPreTag: '<mark>',
  highlightPostTag: '</mark>',
  attributesToCrop: ['description'],
  cropLength: 200,
});

console.log(results.hits);               // matched documents
console.log(results.facetDistribution); // { brand: { Apple: 5, Lenovo: 3 }, ... }
console.log(results.estimatedTotalHits);
```

## POST /indexes/{index}/search — REST API

```bash
curl -X POST "${MEILISEARCH_HOST}/indexes/products/search" \
  -H "Authorization: Bearer ${MEILISEARCH_API_KEY}" \
  -H "Content-Type: application/json" \
  -d '{
    "q": "laptop",
    "filter": "category = laptops AND price < 2000",
    "limit": 10,
    "facets": ["brand"],
    "sort": ["price:asc"]
  }'
```

## Multi-Search Batch

```javascript
// Execute multiple searches in one HTTP request
const results = await client.multiSearch({
  queries: [
    { indexUid: 'products', q: 'laptop',  filter: 'category = laptops' },
    { indexUid: 'products', q: 'monitor', filter: 'category = monitors' },
    { indexUid: 'brands',   q: 'Apple' },
  ],
});

// results.results[0] → laptop search results
// results.results[1] → monitor search results
// results.results[2] → brand search results
```

## Hybrid Search (Semantic + Keyword)

```javascript
// Configure embedder (OpenAI, HuggingFace, or custom REST endpoint)
await index.updateSettings({
  embedders: {
    openai: {
      source: 'openAi',
      apiKey: process.env.OPENAI_API_KEY,
      model: 'text-embedding-3-small',
      documentTemplate: '{{doc.name}} {{doc.description}}',
    },
  },
});

// Hybrid search: combines BM25 keyword + vector semantic search
const results = await index.search('comfortable wireless audio', {
  hybrid: {
    semanticRatio: 0.5,    // 0 = pure keyword, 1 = pure vector, 0.5 = balanced
    embedder: 'openai',
  },
  limit: 20,
});
```

## Facet Search

```javascript
// Search within a specific facet's values
const facetResults = await index.searchForFacetValues({
  facetName: 'brand',
  facetQuery: 'app',       // finds brands matching "app" → Apple, AppTech, etc.
  q: 'laptop',             // constrained to docs matching "laptop"
  filter: 'active = true',
});

console.log(facetResults.facetHits);
// [{ value: 'Apple', count: 5 }, { value: 'AppTech', count: 1 }]
```

## Ranking Rules Customization

```javascript
// Custom ranking: boost products with high rating
await index.updateRankingRules([
  'words',
  'typo',
  'proximity',
  'attribute',
  'sort',
  'exactness',
  'rating:desc',  // custom ranking: higher rating ranks first within same score
]);

// Placeholder search (empty q) respects sort + filters
const featured = await index.search('', {
  sort: ['featured:desc', 'price:asc'],
  filter: 'active = true AND stock > 0',
  limit: 20,
});
```

## Key Rules

- Documents must have a primary key field (specified at index creation or inferred as `id`)
- All Meilisearch operations that modify data are async — always `waitForTask()` in tests
- `filterableAttributes` must be configured before filtering on a field — does not apply retroactively
- Typo tolerance is applied automatically — no query-side configuration needed for basic use
- Meilisearch is not a replacement for Elasticsearch for analytics/aggregations — it lacks complex aggregations
- Use tenant tokens (scoped API keys) to restrict each user to their own data: `client.generateTenantToken(uid, filter)`
- Index size grows with the number of searchable attributes — limit `searchableAttributes` to needed fields
- For large catalogs (>10M docs) use Meilisearch Cloud or self-hosted with NVMe storage
