# Neo4j & ArangoDB Skill Guide

## Overview

Neo4j is the leading native graph database using Cypher as its query language. ArangoDB is a multi-model database supporting documents, graphs, and key-value in a single engine using AQL.

## Neo4j Setup & Connection

```javascript
// Node.js — official driver
import neo4j from 'neo4j-driver';

const driver = neo4j.driver(
  process.env.NEO4J_URI,       // bolt://neo4j:7687 or neo4j+s://...
  neo4j.auth.basic(process.env.NEO4J_USER, process.env.NEO4J_PASSWORD),
  { maxConnectionPoolSize: 50 }
);

// Execute queries via sessions
async function runQuery(cypher, params = {}) {
  const session = driver.session({ database: 'neo4j' });
  try {
    const result = await session.run(cypher, params);
    return result.records;
  } finally {
    await session.close();
  }
}

// Graceful shutdown
process.on('SIGTERM', () => driver.close());
```

## Cypher: Node and Relationship Creation

```cypher
// Create nodes
CREATE (alice:Person {id: 'alice', name: 'Alice', age: 30, city: 'NYC'})
CREATE (bob:Person   {id: 'bob',   name: 'Bob',   age: 25, city: 'LA'})
CREATE (acme:Company {id: 'acme',  name: 'Acme Corp', industry: 'Tech'})

// Create relationships
MATCH (a:Person {id: 'alice'}), (b:Person {id: 'bob'})
CREATE (a)-[:KNOWS {since: 2020, strength: 0.8}]->(b)

MATCH (a:Person {id: 'alice'}), (c:Company {id: 'acme'})
CREATE (a)-[:WORKS_AT {role: 'Engineer', since: 2022}]->(c)

// Merge — upsert node (create only if not exists)
MERGE (p:Person {id: $id})
ON CREATE SET p.name = $name, p.createdAt = datetime()
ON MATCH  SET p.name = $name, p.updatedAt = datetime()
```

## Cypher: MATCH / WHERE / RETURN

```cypher
// Traverse graph: find people Alice knows who work in Tech companies
MATCH (alice:Person {name: 'Alice'})-[:KNOWS]->(colleague:Person)-[:WORKS_AT]->(company:Company)
WHERE company.industry = 'Tech'
  AND colleague.city <> alice.city
RETURN colleague.name AS name, company.name AS employer
ORDER BY colleague.name;

// Variable-length path: find anyone reachable within 3 hops
MATCH (alice:Person {name: 'Alice'})-[:KNOWS*1..3]->(target:Person)
WHERE target <> alice
RETURN DISTINCT target.name, length(path) AS hops;

// Shortest path
MATCH path = shortestPath(
  (alice:Person {name: 'Alice'})-[:KNOWS*]-(bob:Person {name: 'Bob'})
)
RETURN [n IN nodes(path) | n.name] AS connectionPath;

// Aggregation
MATCH (p:Person)-[:WORKS_AT]->(c:Company)
RETURN c.name AS company, count(p) AS headcount, collect(p.name) AS employees
ORDER BY headcount DESC;

// Pattern comprehension
MATCH (p:Person {name: 'Alice'})
RETURN [(p)-[:KNOWS]->(friend) | friend.name] AS friends;
```

## Neo4j Indexes

```cypher
// Range index (default — for equality and range queries)
CREATE INDEX person_id FOR (p:Person) ON (p.id);

// Composite index
CREATE INDEX order_lookup FOR (o:Order) ON (o.userId, o.status);

// Full-text index (for text search)
CREATE FULLTEXT INDEX post_search FOR (p:Post) ON EACH [p.title, p.body];
CALL db.index.fulltext.queryNodes('post_search', 'distributed systems')
  YIELD node, score RETURN node.title, score;

// Point index (for geospatial)
CREATE POINT INDEX location_idx FOR (p:Place) ON (p.location);

// List indexes
SHOW INDEXES;
```

## Neo4j APOC Procedures

```cypher
// APOC: run periodic commit for large imports
CALL apoc.periodic.iterate(
  "UNWIND range(1, 1000000) AS id RETURN id",
  "CREATE (:Node {id: id})",
  {batchSize: 10000, parallel: false}
)
YIELD batches, total;

// APOC: load JSON from URL
CALL apoc.load.json('https://api.example.com/data')
YIELD value
MERGE (p:Product {id: value.id})
SET p.name = value.name;

// APOC: graph refactoring
CALL apoc.refactor.mergeNodes([node1, node2], {properties: 'combine'});
```

## ArangoDB Setup & Connection

```javascript
import { Database } from 'arangojs';

const db = new Database({
  url: process.env.ARANGODB_URL,
  databaseName: process.env.ARANGODB_DATABASE,
  auth: { username: process.env.ARANGODB_USER, password: process.env.ARANGODB_PASSWORD },
});

// Collections
const users = db.collection('users');
const products = db.collection('products');
const follows = db.collection('follows');  // edge collection: _from, _to required
```

## ArangoDB: AQL Graph Traversal

```javascript
// Define graph with vertex and edge collections
await db.createGraph('social', [{
  collection: 'follows',              // edge collection
  from: ['users'],                    // allowed source vertex collections
  to:   ['users'],                    // allowed target vertex collections
}]);

// AQL graph traversal — find users followed by Alice (outbound, 1-3 hops)
const cursor = await db.query(aql`
  FOR v, e, p IN 1..3 OUTBOUND 'users/alice' GRAPH 'social'
    FILTER v.active == true
    SORT LENGTH(p.edges) ASC
    LIMIT 50
    RETURN {
      user: v,
      hops: LENGTH(p.edges),
      path: [FOR n IN p.vertices RETURN n.name]
    }
`);
const results = await cursor.all();

// Shortest path in AQL
const cursor = await db.query(aql`
  FOR p IN OUTBOUND SHORTEST_PATH 'users/alice' TO 'users/bob' GRAPH 'social'
    RETURN p
`);
```

## ArangoDB: Edge Collection with _from / _to

```javascript
// Edge document structure — _from and _to are ArangoDB document handles
await follows.save({
  _from: 'users/alice',       // format: collectionName/documentKey
  _to:   'users/bob',
  followedAt: new Date().toISOString(),
  weight: 0.9,
});

// Insert vertex + connect in one transaction
await db.transaction(
  { write: ['users', 'follows'] },
  async (step) => {
    const user = await step(() => users.save({ name: 'Charlie', active: true }));
    await step(() => follows.save({ _from: 'users/alice', _to: user._id, followedAt: new Date() }));
  }
);
```

## ArangoDB: Multi-Model Queries

```javascript
// Document query (same as a regular document DB)
const cursor = await db.query(aql`
  FOR u IN users
    FILTER u.active == true AND u.country == 'US'
    SORT u.name ASC
    LIMIT 20
    RETURN KEEP(u, '_key', 'name', 'email')
`);

// Graph + Document join — get post details for followed users' posts
const cursor = await db.query(aql`
  FOR followed IN 1..1 OUTBOUND 'users/alice' GRAPH 'social'
    FOR post IN posts
      FILTER post.authorId == followed._key
        AND DATE_DIFF(post.createdAt, DATE_NOW(), 'd') <= 7
      SORT post.createdAt DESC
      LIMIT 50
      RETURN MERGE(post, { author: followed.name })
`);
```

## Key Rules

- Neo4j: avoid `MATCH (n)` without labels — it scans all nodes; always specify label
- Neo4j: use parameters (`$param`) instead of string interpolation to enable query plan caching
- Neo4j: relationships are directional — create in both directions if traversal is bidirectional
- ArangoDB: edge collection documents MUST have `_from` and `_to` properties (format: `collection/key`)
- ArangoDB: AQL GRAPH traversal is significantly faster than manual JOINs for graph queries
- ArangoDB: use `COLLECT` for grouping, not `DISTINCT` — COLLECT is more expressive
- Both: index properties used in FILTER clauses — otherwise full collection scans occur
- Both: avoid `*` (unbounded path) in production — add an upper depth limit like `1..5`
