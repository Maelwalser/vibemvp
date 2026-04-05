# Algolia Skill Guide

## Overview

Algolia is a hosted search and discovery platform with sub-10ms search latency, instant typo tolerance, and a rich ecosystem of InstantSearch UI libraries. It focuses on developer experience and relevance out of the box.

## Index Configuration

```javascript
import algoliasearch from 'algoliasearch';

const client = algoliasearch(process.env.ALGOLIA_APP_ID, process.env.ALGOLIA_ADMIN_KEY);
const index = client.initIndex('products');

// Configure index settings
await index.setSettings({
  // Ranked by relevance weight (order matters)
  searchableAttributes: [
    'unordered(name)',         // unordered = all positions weighted equally
    'brand',
    'description',
    'unordered(tags)',
  ],

  // Faceting and filtering
  attributesForFaceting: [
    'brand',
    'category',
    'searchable(tags)',        // searchable = allows facet search within this attribute
    'filterOnly(active)',      // filterOnly = not shown in UI but can be filtered on
    'filterOnly(userId)',      // used for user-scoped filtering
  ],

  // Custom ranking signals (applied after relevance)
  customRanking: [
    'desc(rating)',            // higher rating wins ties
    'desc(popularity)',        // then more popular
    'asc(price)',              // then cheaper
  ],

  // Default ranking formula (order is significant)
  ranking: [
    'typo',
    'geo',
    'words',
    'filters',
    'proximity',
    'attribute',
    'exact',
    'custom',
  ],

  // Highlighting
  attributesToHighlight: ['name', 'description'],
  attributesToSnippet: ['description:30'],   // 30-word snippet

  // Pagination
  hitsPerPage: 20,
  paginationLimitedTo: 1000,

  // Relevance tuning
  queryLanguages: ['en'],
  removeStopWords: true,
  ignorePlurals: true,
  typoTolerance: 'min',  // 'true' | 'false' | 'min' | 'strict'
});
```

## Record Structure with objectID

```javascript
// objectID is the primary key — must be unique per record
const product = {
  objectID: 'prod-123',    // required — Algolia uses this for deduplication
  name: 'MacBook Pro 14"',
  brand: 'Apple',
  category: 'laptops',
  tags: ['portable', 'professional'],
  price: 1999.00,
  rating: 4.8,
  popularity: 15420,
  active: true,
  userId: 'user-alice',   // for multi-tenant filtering
  _geoloc: { lat: 37.7749, lng: -122.4194 },  // geo search
};
```

## Indexing: saveObjects / partialUpdateObjects

```javascript
// saveObjects — replace full records (upsert by objectID)
const { objectIDs } = await index.saveObjects([product1, product2, product3]);

// partialUpdateObjects — update only specified fields (non-destructive)
await index.partialUpdateObjects([
  { objectID: 'prod-123', price: 1799, rating: 4.9 },
  { objectID: 'prod-124', active: false },
]);

// partialUpdateObjects with createIfNotExists: false — update only, never create
await index.partialUpdateObjects([{ objectID: 'prod-123', stock: 0 }], {
  createIfNotExists: false,
});

// deleteObjects
await index.deleteObjects(['prod-123', 'prod-456']);

// deleteBy — delete matching records
await index.deleteBy({ filters: 'active=false AND updatedAt < 1700000000' });

// Batch operations — mixable actions
await client.multipleBatch([
  { action: 'addObject',     indexName: 'products', body: product },
  { action: 'updateObject',  indexName: 'products', body: { objectID: 'x', price: 10 } },
  { action: 'deleteObject',  indexName: 'products', body: { objectID: 'y' } },
]);
```

## InstantSearch.js Widgets

```javascript
// React InstantSearch
import { InstantSearch, SearchBox, Hits, RefinementList, Pagination, Configure, SortBy } from 'react-instantsearch';
import algoliasearch from 'algoliasearch/lite';

const searchClient = algoliasearch(APP_ID, SEARCH_ONLY_API_KEY);

function ProductSearch() {
  return (
    <InstantSearch searchClient={searchClient} indexName="products">
      <Configure hitsPerPage={20} filters="active:true" />

      <SearchBox placeholder="Search products..." />

      <div className="layout">
        <aside>
          <RefinementList attribute="brand" limit={10} showMore />
          <RefinementList attribute="category" />
          <RangeInput attribute="price" />
        </aside>

        <main>
          <SortBy items={[
            { label: 'Relevance',  value: 'products' },
            { label: 'Price ↑',    value: 'products_price_asc' },
            { label: 'Price ↓',    value: 'products_price_desc' },
          ]} />

          <Hits hitComponent={ProductHit} />
          <Pagination />
        </main>
      </div>
    </InstantSearch>
  );
}

function ProductHit({ hit }) {
  return (
    <article>
      <Highlight attribute="name" hit={hit} />
      <Snippet attribute="description" hit={hit} />
      <span>${hit.price}</span>
    </article>
  );
}
```

## Server-Side Search (API)

```javascript
const results = await index.search('laptop', {
  filters: 'active:true AND price:50 TO 2000 AND category:laptops',
  facets: ['brand', 'category'],
  attributesToRetrieve: ['objectID', 'name', 'price', 'brand', 'rating'],
  hitsPerPage: 20,
  page: 0,
  aroundLatLng: '37.7749,-122.4194',
  aroundRadius: 10000,   // 10km in meters
  getRankingInfo: true,  // include ranking details (dev/debugging)
});
```

## Relevance Tuning with Query Rules

```javascript
// Query Rules — conditional boosts and redirects
await index.saveRule({
  objectID: 'boost-sale-items',
  conditions: [{ pattern: '{facet:category}', anchoring: 'contains' }],
  consequence: {
    params: {
      optionalFilters: 'onSale:true<score=10>',  // boost sale items
    },
    filterPromotes: true,
  },
});

// Redirect rule — send "help" queries to a support page
await index.saveRule({
  objectID: 'redirect-help',
  conditions: [{ pattern: 'help', anchoring: 'is' }],
  consequence: {
    redirect: { url: 'https://support.example.com' },
  },
});

// Pin specific results — merchandise specific products for a query
await index.saveRule({
  objectID: 'pin-macbook-for-apple',
  conditions: [{ pattern: 'apple laptop', anchoring: 'contains' }],
  consequence: {
    promote: [{ objectID: 'prod-macbook-pro', position: 0 }],
  },
});
```

## Synonym Sets

```javascript
await index.saveSynonyms([
  // Bidirectional synonyms
  { objectID: 'laptop-synonyms',   type: 'synonym',     synonyms: ['laptop', 'notebook', 'macbook'] },
  { objectID: 'color-synonyms',    type: 'synonym',     synonyms: ['grey', 'gray', 'charcoal'] },

  // One-way synonym (altCorrection): "iphone" also matches "smartphone"
  { objectID: 'iphone-alt',  type: 'altCorrection1', word: 'iphone',  corrections: ['smartphone'] },

  // Placeholder synonym: "$brand" matches actual brand names
  { objectID: 'brand-placeholder', type: 'placeholder', placeholder: '<brand>',
    replacements: ['Apple', 'Samsung', 'Sony'] },
]);
```

## Multi-Index Search (Federated)

```javascript
// Search multiple indexes in one request
const { results } = await client.multipleQueries([
  { indexName: 'products', query: 'apple', params: { hitsPerPage: 5 } },
  { indexName: 'articles', query: 'apple', params: { hitsPerPage: 3 } },
  { indexName: 'brands',   query: 'apple', params: { hitsPerPage: 3 } },
]);

// results[0] = product hits, results[1] = article hits, results[2] = brand hits
```

## Secured API Keys (Multi-Tenant)

```javascript
// Generate a search-only key scoped to a specific user
const searchKey = client.generateSecuredApiKey(
  process.env.ALGOLIA_SEARCH_KEY,
  {
    filters: `userId:${currentUserId}`,   // enforce data isolation
    validUntil: Math.floor(Date.now() / 1000) + 3600,  // 1 hour expiry
    restrictIndices: ['products'],
  }
);
// Send searchKey to frontend — never the admin key
```

## Key Rules

- `objectID` must be deterministic and unique — derive it from your database primary key
- `searchableAttributes` order determines relevance weight — most important fields first
- `attributesForFaceting` must include all fields used in `filters` parameter
- Use `partialUpdateObjects` for incremental updates — avoids full re-indexing
- Never expose the Admin API key to the frontend — generate secured keys server-side
- Algolia charges per operation — batch all writes and use `saveObjects` not individual saves
- `customRanking` only breaks ties after textual relevance — it does not override Algolia's ranking
- Index replicas are needed for different sort orders (each sort order = one replica index)
