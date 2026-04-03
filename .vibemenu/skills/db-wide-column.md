# Cassandra / ScyllaDB Skill Guide

## Overview

Cassandra and ScyllaDB are wide-column distributed databases optimized for high write throughput and linear horizontal scalability. ScyllaDB is a high-performance C++ reimplementation of Cassandra with the same CQL interface.

## Setup & Connection

```javascript
// Node.js — cassandra-driver
import { Client } from 'cassandra-driver';

const client = new Client({
  contactPoints: ['cassandra-0', 'cassandra-1', 'cassandra-2'],
  localDataCenter: 'datacenter1',
  credentials: { username: process.env.CASSANDRA_USER, password: process.env.CASSANDRA_PASS },
  keyspace: 'myapp',
  pooling: { coreConnectionsPerHost: { local: 2, remote: 1 } },
});
await client.connect();
```

```go
// Go — gocql
import "github.com/gocql/gocql"

cluster := gocql.NewCluster("cassandra-0", "cassandra-1", "cassandra-2")
cluster.Keyspace = "myapp"
cluster.Consistency = gocql.LocalQuorum
cluster.ProtoVersion = 4
cluster.Authenticator = gocql.PasswordAuthenticator{
    Username: os.Getenv("CASSANDRA_USER"),
    Password: os.Getenv("CASSANDRA_PASS"),
}

session, err := cluster.CreateSession()
defer session.Close()
```

## Keyspace Creation with Replication

```cql
-- NetworkTopologyStrategy for multi-datacenter (production)
CREATE KEYSPACE myapp
WITH replication = {
  'class': 'NetworkTopologyStrategy',
  'datacenter1': 3,   -- replication factor per DC
  'datacenter2': 3
}
AND durable_writes = true;

-- SimpleStrategy for single-datacenter / development only
CREATE KEYSPACE myapp_dev
WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};
```

## Primary Key Design

```cql
-- PRIMARY KEY = (partition_key) or ((partition_key), clustering_col1, clustering_col2)
-- Partition key determines which node stores the data
-- Clustering columns determine sort order within a partition

-- Example: messages by conversation
CREATE TABLE messages (
    conversation_id uuid,         -- partition key — all messages for a conversation on one node
    sent_at         timestamp,    -- clustering col — sorted newest-first within partition
    message_id      uuid,
    sender_id       uuid,
    content         text,
    PRIMARY KEY (conversation_id, sent_at, message_id)
) WITH CLUSTERING ORDER BY (sent_at DESC, message_id ASC)
  AND compaction = {'class': 'TimeWindowCompactionStrategy',
                    'compaction_window_unit': 'DAYS',
                    'compaction_window_size': 1};

-- Composite partition key — spreads hot data across nodes
CREATE TABLE metrics (
    metric_name text,
    shard       int,              -- bucket = hash(metric_name) % num_shards
    ts          timestamp,
    value       double,
    PRIMARY KEY ((metric_name, shard), ts)   -- composite partition key
) WITH CLUSTERING ORDER BY (ts DESC);
```

## Tombstones and Compaction Strategies

```cql
-- Tombstones are created by DELETE and TTL expiry
-- Excessive tombstones slow reads — design to avoid deletes where possible

-- Use TTL to auto-expire data (preferred over DELETE for time-series)
INSERT INTO sessions (session_id, user_id, data) VALUES (?, ?, ?) USING TTL 3600;

-- Compaction strategies:
-- SizeTieredCompactionStrategy (STCS) — write-heavy workloads (default)
-- LeveledCompactionStrategy (LCS)     — read-heavy, low tombstone workloads
-- TimeWindowCompactionStrategy (TWCS) — time-series, expired data cleanup

ALTER TABLE events
  WITH compaction = {
    'class': 'TimeWindowCompactionStrategy',
    'compaction_window_unit': 'HOURS',
    'compaction_window_size': 1
  };

-- Monitor tombstones
-- nodetool tablestats myapp.events | grep Tombstone
```

## Consistency Levels

```
Level           Nodes Required   Trade-off
──────────────────────────────────────────────────────────────────────
ONE             1 node           Fastest, may read stale data
LOCAL_ONE       1 node (local DC) Avoids cross-DC latency
QUORUM          (RF/2)+1 nodes   Strong consistency (cross-DC)
LOCAL_QUORUM    (RF/2)+1 (local) Strong within DC, no cross-DC latency (recommended)
ALL             All replicas     Strongest, any failure = error
TWO / THREE     Literal count    Rarely used
EACH_QUORUM     Quorum per DC    For active-active multi-DC writes
```

```javascript
// Per-query consistency level
await client.execute(
  'SELECT * FROM orders WHERE order_id = ?',
  [orderId],
  { consistency: cassandra.types.consistencies.localQuorum }
);
```

## Batch Operations

```cql
-- Logged batch — atomic (all or nothing), same partition key only for performance
BEGIN BATCH
  INSERT INTO orders (order_id, user_id, status) VALUES (uuid(), 'alice', 'pending');
  UPDATE users SET order_count = order_count + 1 WHERE user_id = 'alice';
APPLY BATCH;

-- Unlogged batch — no atomicity guarantee, same partition = efficient
BEGIN UNLOGGED BATCH
  INSERT INTO metrics (metric_name, shard, ts, value) VALUES ('cpu', 0, now(), 0.75);
  INSERT INTO metrics (metric_name, shard, ts, value) VALUES ('cpu', 0, now(), 0.80);
APPLY BATCH;
```

## Secondary Indexes Limitations

```cql
-- Native secondary indexes: only for low-cardinality columns
-- AVOID on high-cardinality columns (UUIDs, emails) — causes full cluster scan

CREATE INDEX ON users (status);  -- OK: low cardinality (active/inactive/banned)
-- BAD: CREATE INDEX ON orders (user_id);  -- use materialized view instead

-- Materialized View (better than secondary index)
CREATE MATERIALIZED VIEW orders_by_user AS
  SELECT * FROM orders
  WHERE user_id IS NOT NULL AND order_id IS NOT NULL
  PRIMARY KEY (user_id, order_id);

-- SASI index (ScyllaDB/Cassandra 3.4+) — for range and text searches
CREATE CUSTOM INDEX ON users (name) USING 'org.apache.cassandra.index.sasi.SASIIndex'
WITH OPTIONS = {'mode': 'CONTAINS'};
```

## Data Modeling Pattern: Query-First

```cql
-- In Cassandra/ScyllaDB, design tables around access patterns, not normalized schema

-- Access pattern 1: Get orders by user (paginated, newest first)
CREATE TABLE orders_by_user (
    user_id    uuid,
    created_at timestamp,
    order_id   uuid,
    total      decimal,
    status     text,
    PRIMARY KEY (user_id, created_at, order_id)
) WITH CLUSTERING ORDER BY (created_at DESC, order_id ASC);

-- Access pattern 2: Get order by ID
CREATE TABLE orders (
    order_id   uuid PRIMARY KEY,
    user_id    uuid,
    total      decimal,
    status     text,
    created_at timestamp
);

-- Write to BOTH tables (denormalization is intentional)
```

## Key Rules

- Never use `SELECT *` without a partition key — results in a full cluster scan
- Partition size should stay under 100 MB; if larger, add a bucketing column
- Use `ALLOW FILTERING` only in development — it causes full-partition scans in production
- Cassandra has no JOINs — denormalize and duplicate data across tables for each query pattern
- Avoid `UPDATE` on primary key columns — delete the old row and insert a new one
- `LOCAL_QUORUM` is the recommended consistency for most production writes/reads in multi-DC
- Monitor with: `nodetool status`, `nodetool tpstats`, `nodetool compactionstats`
- ScyllaDB drops-in for Cassandra — use the same CQL driver and connection strings
