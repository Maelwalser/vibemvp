# PostgreSQL Full-Text Search Skill Guide

## Overview

PostgreSQL has a built-in full-text search engine using `tsvector` (indexed document representation) and `tsquery` (search query). Combined with GIN indexes and `pg_trgm`, it can replace dedicated search engines for moderate workloads.

## Core Types

```sql
-- tsvector: preprocessed, normalized tokens (lexemes)
SELECT to_tsvector('english', 'The quick brown fox jumps over the lazy dogs');
-- 'brown':3 'dog':9 'fox':4 'jump':5 'lazy':8 'quick':2

-- tsquery: boolean search expression
SELECT to_tsquery('english', 'quick & fox');             -- AND
SELECT to_tsquery('english', 'quick | slow');             -- OR
SELECT to_tsquery('english', '!fox & dog');               -- NOT
SELECT to_tsquery('english', 'jump:*');                   -- prefix match
SELECT websearch_to_tsquery('english', 'quick fox -lazy'); -- user-friendly input
SELECT plainto_tsquery('english', 'quick brown fox');      -- phrase → AND query

-- Match operator: @@
SELECT to_tsvector('english', 'fast brown fox') @@ to_tsquery('english', 'fox & fast');
-- → true
```

## Schema Setup

```sql
-- Add tsvector column (auto-updated via trigger or generated column)
ALTER TABLE posts
  ADD COLUMN ts_doc tsvector
  GENERATED ALWAYS AS (
    setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('english', coalesce(body, '')), 'B') ||
    setweight(to_tsvector('english', coalesce(tags::text, '')), 'C')
  ) STORED;

-- GIN index for FTS
CREATE INDEX idx_posts_fts ON posts USING gin(ts_doc);

-- GIN with fastupdate=off: slower writes, faster index (for write-heavy tables)
CREATE INDEX idx_posts_fts ON posts USING gin(ts_doc) WITH (fastupdate = off);
```

## Indexing with to_tsvector

```sql
-- Inline (no stored column) — works but hits the index only with functional index
CREATE INDEX idx_posts_title_fts ON posts USING gin(to_tsvector('english', title));

-- Query must match exactly: same function call
SELECT * FROM posts
WHERE to_tsvector('english', title) @@ to_tsquery('english', 'search');

-- Better: use stored column approach (see Schema Setup above)
SELECT * FROM posts WHERE ts_doc @@ to_tsquery('english', 'database & performance');
```

## websearch_to_tsquery for User Input

```sql
-- websearch_to_tsquery handles natural user input safely (no injection risk)
-- "quick fox"   → 'quick' <-> 'fox'  (phrase)
-- quick fox     → 'quick' & 'fox'    (AND)
-- quick -fox    → 'quick' & !'fox'   (NOT)
-- quick OR fox  → 'quick' | 'fox'    (OR)

SELECT id, title, ts_rank(ts_doc, query) AS rank
FROM posts, websearch_to_tsquery('english', $1) query
WHERE ts_doc @@ query
ORDER BY rank DESC
LIMIT 20;
```

## setweight for Column Weighting

```sql
-- Weights: A (highest) → D (lowest)
-- Used to boost matches in title vs body vs metadata

SELECT setweight(to_tsvector('english', title), 'A')       -- title matches score highest
    || setweight(to_tsvector('english', body), 'B')         -- body matches score lower
    || setweight(to_tsvector('english', author_bio), 'C')   -- bio even lower
    || setweight(to_tsvector('english', tags::text), 'D')   -- tags lowest
AS doc_vector;

-- ts_rank respects weights automatically
SELECT ts_rank(ts_doc, query) FROM posts, websearch_to_tsquery('english', 'postgres') query;
```

## ts_rank and ts_rank_cd Scoring

```sql
-- ts_rank — ranks by frequency of matching lexemes
SELECT
    id,
    title,
    ts_rank(ts_doc, query)    AS rank_freq,     -- frequency-based
    ts_rank_cd(ts_doc, query) AS rank_coverage   -- coverage-based (prefers covering more of query)
FROM posts, websearch_to_tsquery('english', 'distributed database systems') query
WHERE ts_doc @@ query
ORDER BY rank_freq DESC
LIMIT 20;

-- Normalize rank by document length (rank can be skewed for long docs)
-- ts_rank(vector, query, normalization)
-- 0 = no normalization (default)
-- 1 = divide by 1 + log(length)
-- 2 = divide by document length
-- 4 = divide by harmonic distance mean
-- 8 = divide by number of unique words
-- 16 = divide by 1 + log(unique words)
SELECT ts_rank(ts_doc, query, 2) AS normalized_rank
FROM posts, to_tsquery('english', 'search') query
WHERE ts_doc @@ query;
```

## ts_headline for Snippet Extraction

```sql
-- Generate highlighted snippet from matching document
SELECT
    title,
    ts_headline(
        'english',
        body,
        websearch_to_tsquery('english', 'distributed systems'),
        'MaxWords=50, MinWords=20, StartSel=<mark>, StopSel=</mark>, HighlightAll=false'
    ) AS snippet
FROM posts
WHERE ts_doc @@ websearch_to_tsquery('english', 'distributed systems')
LIMIT 10;

-- ts_headline options:
-- MaxWords:        max words in snippet (default 35)
-- MinWords:        min words in snippet (default 15)
-- MaxFragments:    number of snippets to return (0 = one contiguous)
-- FragmentDelimiter: separator between fragments (default "...")
-- HighlightAll:    highlight entire document (default false)
-- StartSel/StopSel: highlight markers (default <b></b>)
```

## pg_trgm for Typo Tolerance

```sql
-- Enable extension
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- GIN trigram index (fast similarity + LIKE/ILIKE queries)
CREATE INDEX idx_products_name_trgm ON products USING gin(name gin_trgm_ops);

-- Fuzzy search using similarity operator
SELECT name, similarity(name, 'Postgress') AS sim
FROM products
WHERE name % 'Postgress'  -- % = similarity > pg_trgm.similarity_threshold (default 0.3)
ORDER BY sim DESC
LIMIT 10;

-- Combine FTS and trigram (for "did you mean?" + full-text)
SELECT
    id, title,
    ts_rank(ts_doc, query) AS fts_rank,
    similarity(title, $1)  AS fuzzy_score
FROM posts, websearch_to_tsquery('english', $1) query
WHERE ts_doc @@ query
   OR title % $1           -- fallback trigram match for typos
ORDER BY fts_rank DESC, fuzzy_score DESC
LIMIT 20;

-- LIKE acceleration with trigram index
SELECT * FROM products WHERE name ILIKE '%macbook%';  -- uses GIN trgm index automatically
```

## Multi-Language Support

```sql
-- List available text search configurations
SELECT cfgname FROM pg_ts_config;

-- Use appropriate language config
SELECT to_tsvector('french',   'Les rapides renards bruns');
SELECT to_tsvector('german',   'schnelle braune Füchse');
SELECT to_tsvector('spanish',  'zorro marrón rápido');
SELECT to_tsvector('simple',   'The 123 fox');  -- no stemming/stopwords (for IDs, codes)

-- Dynamic language from row
SELECT to_tsvector(language::regconfig, content)
FROM articles;
```

## Full Example: Search API Query

```sql
-- Parameterized search with pagination
SELECT
    p.id,
    p.title,
    p.created_at,
    ts_rank(p.ts_doc, query) AS rank,
    ts_headline('english', p.body, query,
        'MaxWords=40, StartSel=<em>, StopSel=</em>') AS snippet
FROM posts p, websearch_to_tsquery('english', $1) query
WHERE p.ts_doc @@ query
  AND p.status = 'published'
  AND ($2::uuid IS NULL OR p.category_id = $2)
ORDER BY rank DESC, p.created_at DESC
LIMIT $3 OFFSET $4;
```

## Key Rules

- Store `tsvector` as a GENERATED ALWAYS AS STORED column — avoids maintaining triggers
- GIN index is required for performance — sequential scan on tsvector is slow
- Use `websearch_to_tsquery` for user input — it is safe and handles natural language
- `to_tsquery` requires valid Tsquery syntax — it throws on invalid input (use in tests only)
- `setweight` must be applied at indexing time — weights stored in the tsvector
- `ts_rank_cd` (cover density) is better for long documents; `ts_rank` for short snippets
- Add `pg_trgm` + similarity index for typo-tolerant fallback ("did you mean?")
- For multilingual content, store the document language and use it as the `regconfig` argument
- FTS is not appropriate for fuzzy prefix search (autocomplete) — use `pg_trgm` with `ILIKE` instead
