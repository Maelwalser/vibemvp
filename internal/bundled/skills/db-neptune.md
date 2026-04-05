# Amazon Neptune Skill Guide

## Overview

Amazon Neptune is a fully managed graph database supporting two graph models: Property Graph (via Gremlin/openCypher) and RDF (via SPARQL). Choose Property Graph for application-level graphs and RDF for linked data / knowledge graphs.

## Model Selection

| Dimension | Property Graph (Gremlin/openCypher) | RDF / SPARQL |
|-----------|-------------------------------------|--------------|
| Use case | Social networks, recommendations, fraud detection | Knowledge graphs, ontologies, semantic web |
| Data model | Vertices + Edges with typed properties | Triples: Subject → Predicate → Object |
| Query | Gremlin (traversal) or openCypher | SPARQL 1.1 |
| Schema | Schema-optional | Defined by ontology (OWL/RDFS) |
| Tooling | Tinkerpop ecosystem, Neo4j-compatible | W3C standard, RDF4J, Jena |

## Connection

```javascript
// Gremlin — @aws-sdk/neptune-graph or gremlin NPM package
import gremlin from 'gremlin';

const { Graph } = gremlin.structure;
const { DriverRemoteConnection } = gremlin.driver;
const { statics: __ } = gremlin.process;

const dc = new DriverRemoteConnection(
  `wss://${process.env.NEPTUNE_ENDPOINT}:8182/gremlin`,
  { mimeType: 'application/vnd.gremlin-v2.0+json' }
);

const g = new Graph().traversal().withRemote(dc);
```

```python
# Python — gremlinpython
from gremlin_python.driver import client, serializer

gremlin_client = client.Client(
    f'wss://{NEPTUNE_ENDPOINT}:8182/gremlin',
    'g',
    message_serializer=serializer.GraphSONMessageSerializer()
)
```

## Gremlin Traversal Patterns

```javascript
// Add vertices
await g.addV('Person').property('name', 'Alice').property('age', 30).next();
await g.addV('Person').property('name', 'Bob').property('age', 25).next();
await g.addV('Movie').property('title', 'Inception').property('year', 2010).next();

// Add edges
await g.V().has('name', 'Alice').as('alice')
  .V().has('name', 'Bob')
  .addE('KNOWS').from_('alice').property('since', 2020).next();

await g.V().has('name', 'Alice').as('alice')
  .V().has('title', 'Inception')
  .addE('WATCHED').from_('alice').property('rating', 5).next();

// Traverse: find what Alice's friends have watched
const results = await g.V().has('Person', 'name', 'Alice')
  .out('KNOWS')             // hop to friends
  .out('WATCHED')           // hop to movies they watched
  .dedup()                   // remove duplicates
  .values('title')           // return movie titles
  .toList();

// Path query: find connection between two people
const path = await g.V().has('name', 'Alice')
  .repeat(__.out('KNOWS').simplePath())
  .until(__.has('name', 'Bob'))
  .limit(1)
  .path()
  .by('name')
  .next();
console.log(path.value.objects);  // ['Alice', 'Charlie', 'Bob']

// Aggregation: most popular movies (by watch count)
const popular = await g.V().hasLabel('Movie')
  .project('title', 'watchCount')
  .by('title')
  .by(__.in_('WATCHED').count())
  .order().by(__.select('watchCount'), gremlin.process.order.desc)
  .limit(10)
  .toList();
```

## SPARQL Triple Patterns

```sparql
# Insert RDF triples
PREFIX ex: <http://example.org/>
PREFIX schema: <https://schema.org/>

INSERT DATA {
  ex:alice a schema:Person ;
           schema:name "Alice" ;
           schema:age 30 ;
           schema:knows ex:bob .

  ex:bob a schema:Person ;
         schema:name "Bob" ;
         schema:age 25 .
}

# Query: find all people Alice knows with their names
SELECT ?name ?age
WHERE {
  ex:alice schema:knows ?person .
  ?person schema:name ?name ;
          schema:age ?age .
}
ORDER BY ?name

# Pattern with OPTIONAL (LEFT JOIN equivalent)
SELECT ?s ?p ?o ?label
WHERE {
  ?s ?p ?o .
  OPTIONAL { ?o schema:name ?label }
}
LIMIT 100

# CONSTRUCT — return new RDF graph
CONSTRUCT { ?person schema:name ?name }
WHERE {
  ?person a schema:Person ;
          schema:name ?name .
  FILTER(xsd:integer(?age) > 25)
}
```

## Index Strategy for Vertex Properties

```javascript
// Neptune auto-manages indexes for Gremlin:
// - Vertex label + property = automatic index
// - Indexed lookups: g.V().has('Person', 'email', 'alice@example.com')

// Property indexes can be created explicitly via:
// Neptune Management API or AWS Console → Indexes tab

// For SPARQL: Neptune indexes all SPO, POS, OSP permutations automatically

// Custom index for frequently-queried properties (Neptune 1.3+)
// POST to management endpoint:
// { "action": "addIndex", "properties": ["email"], "label": "Person" }
```

## Bulk Loader from S3

```bash
# Load from S3 CSV (Property Graph format)
curl -X POST https://${NEPTUNE_ENDPOINT}:8182/loader \
  -H 'Content-Type: application/json' \
  -d '{
    "source": "s3://my-bucket/data/",
    "format": "csv",              # or "nquads", "rdfxml", "turtle"
    "iamRoleArn": "arn:aws:iam::123456789012:role/NeptuneLoadRole",
    "region": "us-east-1",
    "failOnError": "FALSE",
    "parallelism": "MEDIUM"
  }'

# Check load status
curl https://${NEPTUNE_ENDPOINT}:8182/loader/{loadId}
```

```csv
# Vertex CSV format
~id,~label,name:String,age:Int,email:String
alice,Person,Alice,30,alice@example.com
bob,Person,Bob,25,bob@example.com

# Edge CSV format
~id,~from,~to,~label,since:Int,weight:Double
e1,alice,bob,KNOWS,2020,0.9
```

## Key Rules

- Neptune is VPC-only — access via bastion host, VPN, or AWS Lambda in the same VPC
- Use parameter bindings in Gremlin — not string concatenation: `g.V().has('name', name)` not `g.V().has('name', '${name}')`
- Gremlin traversals with `.repeat().until()` must have a `.limit()` — unbounded traversals can OOM
- SPARQL: always use `LIMIT` in exploratory queries — unbounded SELECT can scan entire graph
- Neptune ML integrates with Neptune Analytics for GNN-based predictions on the graph
- Enable Neptune Streams (CDC) to capture graph changes for downstream consumers
- openCypher (Cypher-compatible) is available as a third query language since Neptune 1.2
- Vertex IDs are strings — generate UUIDs at application level for portability
