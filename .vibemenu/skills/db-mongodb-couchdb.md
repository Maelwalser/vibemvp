# MongoDB & CouchDB Skill Guide

## Overview

MongoDB is a document database for flexible JSON-like storage with rich query capabilities. CouchDB is an HTTP-native document store using MapReduce views and multi-master replication.

## MongoDB Setup & Connection

```javascript
// Mongoose connection
import mongoose from 'mongoose';

await mongoose.connect(process.env.MONGODB_URI, {
  maxPoolSize: 10,
  serverSelectionTimeoutMS: 5000,
});

// Native driver
import { MongoClient } from 'mongodb';
const client = new MongoClient(process.env.MONGODB_URI);
const db = client.db('myapp');
```

## Schema Design: Embed vs Reference

```javascript
// EMBED when: data is owned by parent, queried together, bounded size
const orderSchema = new mongoose.Schema({
  userId: { type: mongoose.Schema.Types.ObjectId, ref: 'User', required: true },
  items: [{ // embed — always fetched with order
    productId: String,
    name: String,
    qty: Number,
    price: Number,
  }],
  total: { type: Number, required: true },
  createdAt: { type: Date, default: Date.now },
});

// REFERENCE when: shared across documents, large/unbounded, queried independently
const postSchema = new mongoose.Schema({
  authorId: { type: mongoose.Schema.Types.ObjectId, ref: 'User' }, // reference
  tags: [{ type: mongoose.Schema.Types.ObjectId, ref: 'Tag' }],    // reference
  title: { type: String, required: true, maxlength: 200 },
  body: String,
});
```

## Mongoose Schema with Validators

```javascript
const userSchema = new mongoose.Schema({
  email: {
    type: String,
    required: [true, 'Email required'],
    unique: true,
    lowercase: true,
    match: [/^\S+@\S+\.\S+$/, 'Invalid email'],
  },
  age: { type: Number, min: 0, max: 150 },
  role: { type: String, enum: ['user', 'admin', 'editor'], default: 'user' },
  tags: { type: [String], validate: v => v.length <= 10 },
}, { timestamps: true });

// Virtual
userSchema.virtual('fullName').get(function() {
  return `${this.firstName} ${this.lastName}`;
});
```

## Aggregation Pipeline

```javascript
// $match → $group → $lookup → $project
const result = await Order.aggregate([
  { $match: { status: 'completed', createdAt: { $gte: new Date('2024-01-01') } } },
  { $group: {
    _id: '$userId',
    totalSpent: { $sum: '$total' },
    orderCount: { $count: {} },
    lastOrder: { $max: '$createdAt' },
  }},
  { $lookup: {
    from: 'users',
    localField: '_id',
    foreignField: '_id',
    as: 'user',
    pipeline: [{ $project: { name: 1, email: 1 } }],
  }},
  { $unwind: '$user' },
  { $project: {
    _id: 0,
    userId: '$_id',
    name: '$user.name',
    email: '$user.email',
    totalSpent: 1,
    orderCount: 1,
  }},
  { $sort: { totalSpent: -1 } },
  { $limit: 100 },
]);
```

## Index Design

```javascript
// Single field
userSchema.index({ email: 1 }, { unique: true });

// Compound — equality fields first, then range/sort
orderSchema.index({ userId: 1, createdAt: -1 });

// Text search
postSchema.index({ title: 'text', body: 'text' }, { weights: { title: 10, body: 1 } });

// Geospatial — 2dsphere for GeoJSON
locationSchema.index({ coordinates: '2dsphere' });

// Partial — only index active documents
userSchema.index({ email: 1 }, { partialFilterExpression: { active: true } });

// TTL — auto-delete expired docs
sessionSchema.index({ expiresAt: 1 }, { expireAfterSeconds: 0 });
```

## Query Patterns

```javascript
// Projection to limit fields
const users = await User.find({ role: 'admin' }, { name: 1, email: 1, _id: 0 });

// Cursor pagination — avoid skip() for large collections
const page = await Post.find(
  { _id: { $gt: lastSeenId } }
).sort({ _id: 1 }).limit(20);

// findOneAndUpdate with optimistic locking via version key
const updated = await Product.findOneAndUpdate(
  { _id: id, __v: currentVersion },
  { $inc: { stock: -qty }, $inc: { __v: 1 } },
  { new: true, runValidators: true }
);
if (!updated) throw new Error('Concurrent modification detected');

// Bulk write for efficiency
await Order.bulkWrite([
  { updateOne: { filter: { _id: id1 }, update: { $set: { status: 'shipped' } } } },
  { updateOne: { filter: { _id: id2 }, update: { $set: { status: 'shipped' } } } },
]);
```

## Change Streams (CDC)

```javascript
// Watch collection for changes
const changeStream = Order.watch([
  { $match: { 'fullDocument.status': 'completed' } }
], { fullDocument: 'updateLookup' });

changeStream.on('change', async (event) => {
  if (event.operationType === 'insert') {
    await publishEvent('order.created', event.fullDocument);
  } else if (event.operationType === 'update') {
    await publishEvent('order.updated', event.fullDocument);
  }
});

// Resume after restart using resumeToken
const stream = Order.watch([], { resumeAfter: lastResumeToken });
```

## CouchDB: Document Structure

```json
{
  "_id": "user:alice@example.com",
  "_rev": "3-abc123",
  "type": "user",
  "name": "Alice",
  "email": "alice@example.com",
  "createdAt": "2024-01-15T10:00:00Z"
}
```

## CouchDB: MapReduce Views

```javascript
// Design document with map/reduce
const designDoc = {
  _id: '_design/orders',
  views: {
    by_user: {
      map: function(doc) {
        if (doc.type === 'order') {
          emit(doc.userId, doc.total);
        }
      }.toString(),
      reduce: '_sum',  // built-in: _sum, _count, _stats
    },
    by_date: {
      map: function(doc) {
        if (doc.type === 'order') {
          emit([doc.createdAt.slice(0, 10), doc.userId], null);
        }
      }.toString(),
    },
  },
};

// Query: GET /db/_design/orders/_view/by_user?group=true&key="user123"
```

## CouchDB: Replication

```bash
# Replicate to remote (continuous)
curl -X POST http://localhost:5984/_replicator \
  -H 'Content-Type: application/json' \
  -d '{
    "_id": "sync-to-remote",
    "source": "http://localhost:5984/mydb",
    "target": "https://remote-host:5984/mydb",
    "continuous": true,
    "create_target": true
  }'
```

## Key Rules

- Use `_id` prefixes by type (e.g., `user:`, `order:`) in CouchDB for natural sharding
- Embed arrays only when bounded — unbounded arrays cause document bloat
- Index every field used in `$match` or as a join key
- Use `explain()` to verify index usage: `Order.find({...}).explain('executionStats')`
- Change streams require a replica set or sharded cluster
- Avoid `$where` (JS eval) — use `$expr` with aggregation operators instead
- CouchDB: never update `_rev` manually — always fetch current doc before update
