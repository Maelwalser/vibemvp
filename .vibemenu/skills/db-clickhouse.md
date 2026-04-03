# ClickHouse Skill Guide

## Overview

ClickHouse is a column-oriented OLAP database optimized for analytical queries on large datasets. It achieves high throughput via vectorized execution, compression, and a sparse primary index.

## MergeTree Engine Family

```sql
-- MergeTree — base engine for most analytical workloads
CREATE TABLE events (
    event_date  Date,
    event_time  DateTime,
    user_id     String,
    event_type  LowCardinality(String),
    properties  String,           -- JSON as string
    amount      Decimal64(2)
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(event_date)    -- one partition per month
ORDER BY (event_type, user_id, event_time)  -- sparse primary index
SETTINGS index_granularity = 8192;

-- ReplacingMergeTree — idempotent upserts (deduplication at merge time)
CREATE TABLE products (
    updated_at  DateTime,
    product_id  String,
    name        String,
    price       Decimal64(2),
    active      UInt8
) ENGINE = ReplacingMergeTree(updated_at)   -- version column: keep highest
PARTITION BY toYYYYMM(updated_at)
ORDER BY product_id;

-- SummingMergeTree — automatic aggregation of numeric columns
CREATE TABLE daily_revenue (
    date        Date,
    region      LowCardinality(String),
    revenue     Decimal64(2),
    order_count UInt64
) ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (date, region);

-- AggregatingMergeTree — pre-aggregated states (with materialized views)
CREATE TABLE user_stats (
    date           Date,
    user_id        String,
    total_spend    AggregateFunction(sum, Decimal64(2)),
    order_count    AggregateFunction(count, UInt64),
    unique_items   AggregateFunction(uniq, String)
) ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (date, user_id);
```

## PARTITION BY Design

```sql
-- Monthly partitioning (most common for time-series)
PARTITION BY toYYYYMM(event_date)

-- Daily partitioning (high-volume, short-retention data)
PARTITION BY toYYYYMMDD(event_time)

-- Custom expression
PARTITION BY (toYear(event_date), cityHash64(user_id) % 16)  -- year + shard

-- List existing partitions
SELECT partition, name, rows, bytes_on_disk
FROM system.parts
WHERE table = 'events' AND active
ORDER BY partition;

-- Drop old partitions (data lifecycle)
ALTER TABLE events DROP PARTITION 202301;
```

## ORDER BY as Sparse Primary Index

```sql
-- ORDER BY determines the sparse index (not a separate index)
-- Rules:
--   1. Equality-filter columns first (highest cardinality reduces scan)
--   2. Range/sort columns last
--   3. Avoid high-cardinality UUID first (kills compression)

-- GOOD: filter by event_type (low cardinality), then user_id, then time range
ORDER BY (event_type, user_id, event_time)

-- BAD: UUID first — bad compression, poor index pruning
ORDER BY (user_id, event_type, event_time)

-- Check index granule stats
SELECT name, marks, rows_per_mark
FROM system.parts
WHERE table = 'events' AND active;
```

## Materialized Views for Incremental Aggregation

```sql
-- Target table (AggregatingMergeTree)
CREATE TABLE hourly_stats (
    hour        DateTime,
    event_type  LowCardinality(String),
    user_count  AggregateFunction(uniq, String),
    event_count AggregateFunction(count, UInt64),
    total_spend AggregateFunction(sum, Decimal64(2))
) ENGINE = AggregatingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (hour, event_type);

-- Materialized view — triggers on every INSERT to events
CREATE MATERIALIZED VIEW hourly_stats_mv
TO hourly_stats
AS SELECT
    toStartOfHour(event_time) AS hour,
    event_type,
    uniqState(user_id)        AS user_count,
    countState()              AS event_count,
    sumState(amount)          AS total_spend
FROM events
GROUP BY hour, event_type;

-- WITH NO DATA = don't backfill historical data on creation
CREATE MATERIALIZED VIEW hourly_stats_mv TO hourly_stats
WITH NO DATA
AS SELECT ...;

-- Add refresh policy for backfill (ClickHouse 23.8+)
ALTER TABLE hourly_stats_mv REFRESH EVERY 1 HOUR OFFSET 5 MINUTE;

-- Query the materialized view
SELECT
    hour,
    event_type,
    uniqMerge(user_count)   AS users,
    countMerge(event_count) AS events,
    sumMerge(total_spend)   AS revenue
FROM hourly_stats
WHERE hour >= now() - INTERVAL 24 HOUR
GROUP BY hour, event_type
ORDER BY hour DESC;
```

## TTL for Data Lifecycle

```sql
-- Auto-delete rows after 90 days
CREATE TABLE events (
    event_date Date,
    ...
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(event_date)
ORDER BY (event_date, user_id)
TTL event_date + INTERVAL 90 DAY;

-- Move old data to slower disk tier (instead of delete)
TTL event_date + INTERVAL 30 DAY TO DISK 'cold_disk',
    event_date + INTERVAL 90 DAY TO VOLUME 'archive';

-- Alter existing table
ALTER TABLE events MODIFY TTL event_date + INTERVAL 90 DAY;
```

## PREWHERE for Early Filtering

```sql
-- PREWHERE is evaluated before WHERE — reads only the filtered rows' remaining columns
-- ClickHouse auto-converts simple WHERE to PREWHERE for indexed columns
-- Use explicitly for expensive string/LIKE filters

SELECT user_id, sum(amount)
FROM events
PREWHERE event_type = 'purchase'          -- cheap filter first (index)
WHERE amount > 100                         -- secondary filter on loaded data
  AND has(splitByChar(',', tags), 'vip')  -- expensive — evaluated last
GROUP BY user_id;
```

## Compression Codecs

```sql
-- Column-level codec declaration
CREATE TABLE metrics (
    ts          DateTime CODEC(Delta, LZ4),     -- Delta = store diffs (great for timestamps)
    value       Float64  CODEC(Gorilla, LZ4),   -- Gorilla = float compression
    counter     UInt64   CODEC(Delta(8), ZSTD(3)), -- Delta then ZSTD for smaller files
    label       String   CODEC(ZSTD(3)),        -- general string compression
    raw_event   String   CODEC(NONE)            -- disable compression for pre-compressed data
) ENGINE = MergeTree() ...;

-- Codec selection guide:
-- Timestamps/monotonic integers: Delta + LZ4
-- Float64 time-series:           Gorilla + LZ4
-- General strings:               ZSTD(3) or LZ4
-- High-cardinality strings:      LowCardinality(String) (dictionary encoding)
```

## Query Profiling with query_log

```sql
-- Find slow queries (last hour)
SELECT
    query_id,
    query_duration_ms,
    read_rows,
    read_bytes,
    result_rows,
    memory_usage,
    query
FROM system.query_log
WHERE type = 'QueryFinish'
  AND event_time >= now() - INTERVAL 1 HOUR
  AND query_duration_ms > 1000
ORDER BY query_duration_ms DESC
LIMIT 20;

-- Find queries reading too many rows (missing index pruning)
SELECT query, read_rows, result_rows,
       round(read_rows / result_rows) AS read_to_result_ratio
FROM system.query_log
WHERE type = 'QueryFinish'
  AND event_time >= now() - INTERVAL 1 HOUR
  AND read_rows / result_rows > 1000   -- reading 1000x more than returning
ORDER BY read_rows DESC
LIMIT 10;

-- Enable query_log (default: on)
-- SET log_queries = 1;
```

## Key Rules

- Always specify `PARTITION BY` — queries filtering on the partition key skip irrelevant parts
- `ORDER BY` columns must include all `PARTITION BY` columns (or be a subset)
- Use `LowCardinality(String)` for columns with < 10,000 distinct values (status, type, region)
- Batch inserts: minimum 1,000 rows per INSERT; async_insert mode for small producers
- Avoid `FINAL` keyword in hot paths — it forces merge at query time; schedule merges instead
- Never use `DISTINCT` on high-cardinality columns in aggregations — use `uniq()` instead
- `ReplacingMergeTree` deduplication is eventually consistent — use `FINAL` or `GROUP BY` for exact results
- Primary key is sparse (one entry per ~8,192 rows) — not a unique constraint
