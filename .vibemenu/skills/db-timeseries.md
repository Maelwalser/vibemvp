# Time-Series Databases Skill Guide

## Overview

Covers three time-series databases: **TimescaleDB** (PostgreSQL extension for SQL + time-series), **InfluxDB** (purpose-built TSB with Flux), and **QuestDB** (high-performance SQL TSB with SIMD execution).

---

## TimescaleDB

### Setup & Connection

```sql
-- Create extension in PostgreSQL
CREATE EXTENSION IF NOT EXISTS timescaledb;
```

```javascript
// Node.js — standard pg driver
import { Pool } from 'pg';
const pool = new Pool({ connectionString: process.env.DATABASE_URL });
```

### Hypertable Creation

```sql
-- Regular table first, then convert to hypertable
CREATE TABLE metrics (
    ts         TIMESTAMPTZ NOT NULL,
    device_id  TEXT        NOT NULL,
    metric     TEXT        NOT NULL,
    value      DOUBLE PRECISION NOT NULL
);

-- Convert to hypertable (partitions by time automatically)
SELECT create_hypertable('metrics', 'ts',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists       => TRUE
);

-- Create indexes (ts is automatically indexed by hypertable)
CREATE INDEX ON metrics (device_id, ts DESC);
CREATE INDEX ON metrics (metric, ts DESC);
```

### time_bucket() for Aggregation

```sql
-- 1-hour buckets
SELECT
    time_bucket('1 hour', ts) AS bucket,
    device_id,
    AVG(value)   AS avg_value,
    MAX(value)   AS max_value,
    MIN(value)   AS min_value,
    COUNT(*)     AS samples
FROM metrics
WHERE ts >= NOW() - INTERVAL '24 hours'
  AND metric = 'cpu_usage'
GROUP BY bucket, device_id
ORDER BY bucket DESC;

-- Gap-filling: fill missing buckets with NULL or previous value
SELECT
    time_bucket_gapfill('5 minutes', ts) AS bucket,
    device_id,
    locf(AVG(value)) AS value_last_obs_carried_forward
FROM metrics
WHERE ts >= NOW() - INTERVAL '1 hour'
  AND metric = 'temperature'
GROUP BY bucket, device_id;
```

### Continuous Aggregates

```sql
-- Define continuous aggregate (materialized and auto-refreshed)
CREATE MATERIALIZED VIEW hourly_metrics
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 hour', ts) AS bucket,
    device_id,
    metric,
    AVG(value)   AS avg_value,
    MAX(value)   AS max_value,
    COUNT(*)     AS sample_count
FROM metrics
GROUP BY bucket, device_id, metric
WITH NO DATA;  -- don't backfill on creation

-- Add refresh policy (refresh every 30 min, backfill last 2 hours)
SELECT add_continuous_aggregate_policy('hourly_metrics',
    start_offset    => INTERVAL '2 hours',
    end_offset      => INTERVAL '30 minutes',
    schedule_interval => INTERVAL '30 minutes'
);

-- Query continuous aggregate (same as table)
SELECT * FROM hourly_metrics
WHERE bucket >= NOW() - INTERVAL '7 days'
  AND metric = 'cpu_usage'
ORDER BY bucket DESC;
```

### Compression Policy

```sql
-- Enable compression (compresses chunks older than 7 days)
ALTER TABLE metrics SET (timescaledb.compress,
    timescaledb.compress_orderby   = 'ts DESC',
    timescaledb.compress_segmentby = 'device_id, metric'
);

SELECT add_compression_policy('metrics', INTERVAL '7 days');

-- Check compression stats
SELECT * FROM chunk_compression_stats('metrics');
```

---

## InfluxDB

### Measurement / Tag / Field Model

```
Measurement: cpu_usage
Tags (indexed, string):  host=web-1, region=us-east-1
Fields (not indexed):    value=0.75, cores=8
Timestamp:               2024-01-15T10:00:00Z
```

### Connection & Write

```javascript
import { InfluxDB, Point } from '@influxdata/influxdb-client';

const client = new InfluxDB({
  url: process.env.INFLUX_URL,
  token: process.env.INFLUX_TOKEN,
});

const writeApi = client.getWriteApi(process.env.INFLUX_ORG, process.env.INFLUX_BUCKET, 's');

// Write data points
const point = new Point('cpu_usage')
  .tag('host', 'web-1')
  .tag('region', 'us-east-1')
  .floatField('value', 0.75)
  .intField('cores', 8)
  .timestamp(new Date());

writeApi.writePoint(point);
await writeApi.flush();
```

### Flux Query Language

```javascript
const queryApi = client.getQueryApi(process.env.INFLUX_ORG);

const query = `
  from(bucket: "${process.env.INFLUX_BUCKET}")
    |> range(start: -24h)
    |> filter(fn: (r) => r._measurement == "cpu_usage")
    |> filter(fn: (r) => r.host == "web-1")
    |> aggregateWindow(every: 1h, fn: mean, createEmpty: false)
    |> yield(name: "hourly_avg")
`;

const rows = [];
await queryApi.collectRows(query, (row, tableMeta) => {
  rows.push(tableMeta.toObject(row));
});
```

### Bucket Retention Policy

```bash
# Create bucket with 30-day retention
influx bucket create \
  --name my-metrics \
  --org my-org \
  --retention 720h  # 30 days; 0 = infinite
```

---

## QuestDB

### Connection

```javascript
// REST API (HTTP/1.1)
const query = async (sql) => {
  const res = await fetch(`${process.env.QUESTDB_URL}/exec?query=${encodeURIComponent(sql)}`);
  return res.json();
};

// ILP (Influx Line Protocol) for high-throughput ingestion — UDP or TCP port 9009
import { Sender } from '@questdb/nodejs-client';
const sender = Sender.fromConfig(`tcp::addr=${process.env.QUESTDB_HOST}:9009;`);

await sender.table('metrics')
  .symbol('device', 'sensor-1')       // indexed symbol column
  .floatColumn('temperature', 22.4)
  .at(Date.now(), 'ms');              // timestamp

await sender.flush();
await sender.close();
```

### SAMPLE BY for Time Aggregation

```sql
-- SAMPLE BY: automatic time bucketing
SELECT
    ts,
    device,
    avg(temperature) AS avg_temp,
    max(temperature) AS max_temp,
    count()          AS readings
FROM sensor_data
WHERE ts IN '2024-01-01T00:00:00Z' .. '2024-01-15T00:00:00Z'
  AND device = 'sensor-1'
SAMPLE BY 1h FILL(PREV);  -- fill gaps with previous value

-- FILL options: NONE, NULL, PREV, LINEAR, value
```

### LATEST ON for Last-Known Value

```sql
-- Get the most recent reading for each device
SELECT device, temperature, ts
FROM sensor_data
LATEST ON ts PARTITION BY device;

-- Filter + latest
SELECT device, temperature, ts
FROM sensor_data
WHERE temperature > 30.0
LATEST ON ts PARTITION BY device;
```

### ASOF JOIN for Time-Series Correlation

```sql
-- Correlate two time-series at the nearest timestamp
SELECT
    s.ts,
    s.device,
    s.temperature,
    w.humidity,
    w.pressure
FROM sensor_data s
ASOF JOIN weather_data w ON (s.city = w.city);  -- join on nearest-time + key
```

## Key Rules

- TimescaleDB: `chunk_time_interval` should match your common query window (1 day for sub-day queries)
- TimescaleDB: continuous aggregates cannot have row-level security — use views with RLS on top
- InfluxDB: tags are indexed (use for filtering), fields are not (use for measured values)
- InfluxDB: avoid high-cardinality tags (UUIDs, emails) — they balloon the series index
- QuestDB: designated timestamp column must be declared at table creation and cannot be changed
- QuestDB: `SAMPLE BY` requires a designated timestamp; results are ordered by time automatically
- All three: batch writes aggressively — single-row inserts are expensive; use bulk/streaming APIs
- All three: retention and compression policies are mandatory in production to control disk growth
