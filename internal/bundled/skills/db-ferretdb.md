# FerretDB Skill Guide

## Overview

FerretDB is a stateless proxy that translates MongoDB wire protocol to PostgreSQL SQL, storing BSON documents as JSONB. It allows teams to use MongoDB drivers and tooling while storing data in PostgreSQL.

## Connection

```javascript
// Same connection string as MongoDB — just point to FerretDB
const client = new MongoClient('mongodb://user:pass@ferretdb-host:27017/mydb?authMechanism=PLAIN');

// Mongoose
await mongoose.connect('mongodb://user:pass@ferretdb-host:27017/mydb?authMechanism=PLAIN');

// mongosh
mongosh 'mongodb://user:pass@ferretdb-host:27017/mydb?authMechanism=PLAIN'
```

## Kubernetes Deployment (Stateless Proxy)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ferretdb
spec:
  replicas: 3  # stateless — scale horizontally
  selector:
    matchLabels:
      app: ferretdb
  template:
    metadata:
      labels:
        app: ferretdb
    spec:
      containers:
        - name: ferretdb
          image: ghcr.io/ferretdb/ferretdb:latest
          env:
            - name: FERRETDB_POSTGRESQL_URL
              valueFrom:
                secretKeyRef:
                  name: pg-credentials
                  key: url
            - name: FERRETDB_LOG_LEVEL
              value: "info"
          ports:
            - containerPort: 27017
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 512Mi
---
apiVersion: v1
kind: Service
metadata:
  name: ferretdb
spec:
  selector:
    app: ferretdb
  ports:
    - port: 27017
      targetPort: 27017
```

## BSON to JSONB Mapping

```
MongoDB Type     → PostgreSQL JSONB
─────────────────────────────────────
ObjectId         → {"$o": "hexstring"}
Date             → {"$d": milliseconds}
Regex            → {"$r": "pattern", "$ro": "flags"}
Binary           → {"$b": "base64", "$t": subtype}
Int32            → number (native JSON)
Int64            → {"$l": "string"}
Decimal128       → {"$n": "string"}
String           → string (native JSON)
Boolean          → boolean (native JSON)
Array            → array (native JSON)
Document         → object (native JSON)
```

## How FerretDB Stores Data in PostgreSQL

```sql
-- FerretDB creates tables like this internally:
-- Each MongoDB collection → one PostgreSQL table
CREATE TABLE mydb.orders (
    _jsonb JSONB
);

-- FerretDB generates GIN indexes for queries:
CREATE INDEX ON mydb.orders USING gin (_jsonb jsonb_path_ops);

-- Inspect stored documents directly in PostgreSQL:
SELECT _jsonb FROM mydb.orders WHERE _jsonb @> '{"status": "completed"}';
```

## SQL Pushdown Optimization

FerretDB pushes simple filter operations down to PostgreSQL. To benefit:

```javascript
// GOOD — equality and range filters pushed to PostgreSQL JSONB operators
await db.orders.find({ status: 'completed', total: { $gte: 100 } });

// GOOD — $in becomes ANY() in PostgreSQL
await db.orders.find({ status: { $in: ['pending', 'processing'] } });

// LESS EFFICIENT — complex JS $where not supported, use $expr instead
// FerretDB does NOT support $where
await db.orders.find({ $expr: { $gt: ['$total', '$minTotal'] } });
```

## Migration from MongoDB

```bash
# Export from MongoDB
mongodump --uri="mongodb://mongo-host/mydb" --out=/tmp/dump

# Import into FerretDB
mongorestore --uri="mongodb://user:pass@ferretdb-host:27017/mydb?authMechanism=PLAIN" /tmp/dump

# Or stream with mongomirror (for live migration)
mongomirror \
  --from "mongodb://mongo-host:27017" \
  --destination "mongodb://user:pass@ferretdb-host:27017?authMechanism=PLAIN" \
  --namespace "mydb.*"
```

## Limitations vs Native MongoDB

| Feature | FerretDB Support |
|---------|-----------------|
| Change Streams | Not supported |
| Transactions (multi-doc) | Partial — depends on PG version |
| Capped Collections | Not supported |
| GridFS | Not supported |
| $text search | Not supported (use pg_trgm via PostgreSQL directly) |
| Aggregation pipeline | Partial — growing support |
| Atlas Search | Not supported |
| Time Series collections | Not supported |

## Key Rules

- FerretDB is stateless — store all data in PostgreSQL, not in FerretDB pods
- Use `authMechanism=PLAIN` in connection strings — FerretDB delegates auth to PostgreSQL
- Indexes defined via `createIndex()` are translated to PostgreSQL GIN/B-tree indexes
- For unsupported operations, fall back to PostgreSQL directly via `pg` client
- Monitor PostgreSQL JSONB column sizes — large embedded arrays degrade performance
- FerretDB inherits PostgreSQL's ACID guarantees and connection pooling via pgbouncer
- Pin FerretDB version in production — compatibility layer evolves across releases
