# Dgraph Skill Guide

## Overview

Dgraph is a horizontally scalable, distributed native graph database supporting both GraphQL and DQL (Dgraph Query Language). It stores data as RDF-like triples (subject-predicate-object) and provides ACID transactions.

## Setup & Connection

```javascript
// dgraph-js-http (browser & Node.js)
import { DgraphClient, DgraphClientStub, Mutation, Request } from 'dgraph-js-http';

const clientStub = new DgraphClientStub(process.env.DGRAPH_ALPHA_URL); // e.g., http://dgraph-alpha:8080
const client = new DgraphClient(clientStub);

// Drop all data and schema (dev reset only)
// await client.alter({ dropAll: true });
```

```go
// Go — dgo
import (
    "google.golang.org/grpc"
    "github.com/dgraph-io/dgo/v230"
    "github.com/dgraph-io/dgo/v230/protos/api"
)

conn, _ := grpc.Dial("dgraph-alpha:9080", grpc.WithInsecure())
dg := dgo.NewDgraphClient(api.NewDgraphClient(conn))
```

## GraphQL Schema to Predicate Mapping

```graphql
# schema.graphql — deployed to /admin/schema endpoint
type Person {
    id:       ID!
    name:     String!  @search(by: [term, fulltext])
    email:    String   @id          # @id = unique index
    age:      Int      @search
    location: Point    @search(by: [near, within])
    friends:  [Person] @hasInverse(field: friends)
    posts:    [Post]   @hasInverse(field: author)
    createdAt: DateTime @search
}

type Post {
    id:       ID!
    title:    String! @search(by: [term, fulltext])
    body:     String  @search(by: [fulltext])
    author:   Person!
    tags:     [String] @search(by: [term])
    likes:    Int     @search
    createdAt: DateTime @search
}
```

```bash
# Deploy schema via HTTP
curl -X POST http://dgraph-alpha:8080/admin/schema \
  -H 'Content-Type: application/graphql' \
  --data-binary @schema.graphql
```

## DQL: Dgraph Query Language

```javascript
// Basic DQL query — find persons named Alice and their friends
const query = `
{
  findPerson(func: eq(Person.name, "Alice")) {
    uid
    Person.name
    Person.email
    Person.friends {
      uid
      Person.name
      Person.age
    }
    Person.posts(first: 10, orderasc: Post.createdAt) {
      Post.title
      Post.likes
    }
  }
}`;

const res = await client.newTxn({ readOnly: true }).queryWithVars(query, {});
const data = JSON.parse(res.getJson());
```

## DQL: Mutations

```javascript
// JSON-based mutation (recommended)
const mutation = {
  setJson: {
    'dgraph.type': 'Person',
    'Person.name': 'Alice',
    'Person.email': 'alice@example.com',
    'Person.age': 30,
    'Person.friends': [
      { uid: '_:bob' },  // reference existing or blank node
    ],
  },
};

const txn = client.newTxn();
try {
  const response = await txn.mutate({ setJson: mutation.setJson });
  await txn.commit();
  const uid = response.getUidsMap().get('blank-0'); // uid of created node
} catch (err) {
  await txn.discard();
  throw err;
}
```

## @reverse for Bidirectional Edges

```graphql
# In GraphQL schema — @hasInverse creates automatic reverse edges
type Person {
    friends: [Person] @hasInverse(field: friends)  # symmetric
    posts:   [Post]   @hasInverse(field: author)   # author ↔ posts
}
type Post {
    author:  Person!                                # linked back to Person.posts
}
```

```bash
# In DQL schema — define reverse predicate explicitly
Person.friends: [uid] @reverse .
Post.author:    uid   @reverse .
```

## ACID Transactions

```go
// Go — ACID transaction with commit and discard
ctx := context.Background()

txn := dg.NewTxn()
defer txn.Discard(ctx)

// Read within transaction (consistent snapshot)
q := `{ user(func: eq(Person.email, "alice@example.com")) { uid Person.balance } }`
resp, err := txn.Query(ctx, q)

// Parse response, modify data
var result struct { User []struct { UID string `json:"uid"`; Balance float64 `json:"Person.balance"` } `json:"user"` }
json.Unmarshal(resp.Json, &result)

// Mutate within same transaction
mu := &api.Mutation{
    SetJson: []byte(fmt.Sprintf(`{"uid": "%s", "Person.balance": %f}`, result.User[0].UID, newBalance)),
}
_, err = txn.Mutate(ctx, mu)
if err != nil { return err }

// Commit — will fail if another transaction modified the same data
err = txn.Commit(ctx)
```

## upsertBlock for Merge

```javascript
// Upsert: query + conditional mutation in one atomic operation
const upsertReq = {
  query: `{ user as var(func: eq(Person.email, "alice@example.com")) }`,
  mutations: [{
    // Only runs if user found (uid(user) exists)
    cond: '@if(gt(len(user), 0))',
    setJson: { uid: 'uid(user)', 'Person.loginCount': 1 },  // simplified — use @facets for increments
  }, {
    // Only runs if user NOT found
    cond: '@if(eq(len(user), 0))',
    setJson: {
      'dgraph.type': 'Person',
      'Person.email': 'alice@example.com',
      'Person.name':  'Alice',
      'Person.loginCount': 1,
    },
  }],
};

const txn = client.newTxn();
await txn.doRequest({ query: upsertReq.query, mutations: upsertReq.mutations });
await txn.commit();
```

## Super-Node Detection and Mitigation

```javascript
// Super-nodes: predicates with millions of edges (e.g., "is_tagged_with" on a popular tag)
// They slow all queries touching that node.

// Detect: count edges on a suspect node
const query = `{ count(func: uid(0x1)) { count(Person.friends) } }`;

// Mitigations:
// 1. Index reverse predicates to avoid loading the full edge list
// 2. Use @facets to store metadata on edges (filters reduce edge traversal)
// 3. Partition super-nodes: instead of one "tag:javascript" node,
//    use "tag:javascript:2024-01", "tag:javascript:2024-02" etc.
// 4. Use @normalize + @cascade to limit result set before following super-node edges
```

## Key Rules

- Always `defer txn.Discard(ctx)` in Go — ensures cleanup even on error
- Dgraph UIDs are 64-bit hex strings — store as strings in application layer
- `@id` directive creates a unique index on that predicate (like a primary key)
- `@search` directive enables filtering; without it, only `uid` lookups are possible
- GraphQL and DQL share the same underlying data — use GraphQL for CRUD, DQL for analytics
- Avoid loading unbounded predicates: use `first`, `offset`, `after` for pagination
- Dgraph cluster requires Alpha nodes (data) and Zero nodes (coordination) — minimum 1 of each
- Use `dgraph live` loader for bulk imports — REST mutations are too slow for millions of triples
