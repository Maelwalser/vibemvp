# DynamoDB Skill Guide

## Overview

DynamoDB is a fully managed NoSQL key-value and document database. The single-table design pattern collapses multiple entity types into one table using composite keys, enabling efficient access patterns without joins.

## Setup & Connection

```javascript
import { DynamoDBClient } from '@aws-sdk/client-dynamodb';
import { DynamoDBDocumentClient, GetCommand, PutCommand, QueryCommand, TransactWriteCommand } from '@aws-sdk/lib-dynamodb';

const client = new DynamoDBClient({ region: process.env.AWS_REGION });
const ddb = DynamoDBDocumentClient.from(client, {
  marshallOptions: { removeUndefinedValues: true },
});
```

## Single-Table Design

```javascript
// Table: MyApp
// PK = partition key, SK = sort key
// All entity types coexist in one table

// User entity
{ PK: 'USER#alice',      SK: 'PROFILE',         type: 'user',  name: 'Alice', email: 'alice@example.com' }

// Order entity (belongs to user)
{ PK: 'USER#alice',      SK: 'ORDER#2024-01-15#uuid1', type: 'order', total: 99.99, status: 'completed' }

// Product entity
{ PK: 'PRODUCT#prod-1',  SK: 'METADATA',        type: 'product', name: 'Widget', price: 9.99 }

// Access patterns covered:
// Get user profile         → GetItem PK=USER#alice SK=PROFILE
// Get user's orders        → Query PK=USER#alice SK begins_with ORDER#
// Get orders by date range → Query PK=USER#alice SK between ORDER#2024-01 and ORDER#2024-02
```

## Table Creation with GSI and LSI

```javascript
import { CreateTableCommand } from '@aws-sdk/client-dynamodb';

await client.send(new CreateTableCommand({
  TableName: 'MyApp',
  KeySchema: [
    { AttributeName: 'PK', KeyType: 'HASH' },
    { AttributeName: 'SK', KeyType: 'RANGE' },
  ],
  AttributeDefinitions: [
    { AttributeName: 'PK',    AttributeType: 'S' },
    { AttributeName: 'SK',    AttributeType: 'S' },
    { AttributeName: 'GSI1PK', AttributeType: 'S' },
    { AttributeName: 'GSI1SK', AttributeType: 'S' },
  ],
  GlobalSecondaryIndexes: [{
    IndexName: 'GSI1',
    KeySchema: [
      { AttributeName: 'GSI1PK', KeyType: 'HASH' },
      { AttributeName: 'GSI1SK', KeyType: 'RANGE' },
    ],
    Projection: { ProjectionType: 'ALL' },
  }],
  LocalSecondaryIndexes: [{  // LSI must be defined at table creation, same PK
    IndexName: 'LSI1',
    KeySchema: [
      { AttributeName: 'PK', KeyType: 'HASH' },
      { AttributeName: 'GSI1SK', KeyType: 'RANGE' },  // alternate sort key
    ],
    Projection: { ProjectionType: 'INCLUDE', NonKeyAttributes: ['status', 'total'] },
  }],
  BillingMode: 'PAY_PER_REQUEST',  // on-demand
  // or: BillingMode: 'PROVISIONED', ProvisionedThroughput: { ReadCapacityUnits: 5, WriteCapacityUnits: 5 }
}));
```

## GSI for Alternate Access Patterns

```javascript
// GSI1PK = entity type, GSI1SK = status#date — allows querying all orders by status
const completedOrders = await ddb.send(new QueryCommand({
  TableName: 'MyApp',
  IndexName: 'GSI1',
  KeyConditionExpression: 'GSI1PK = :type AND begins_with(GSI1SK, :status)',
  ExpressionAttributeValues: {
    ':type': 'ORDER',
    ':status': 'completed#2024-01',
  },
  ScanIndexForward: false,  // newest first
  Limit: 50,
}));
```

## DynamoDB Streams for Event-Driven

```javascript
// Lambda trigger on stream — process INSERT/MODIFY/REMOVE events
export const handler = async (event) => {
  for (const record of event.Records) {
    const { eventName, dynamodb } = record;
    if (eventName === 'INSERT') {
      const newItem = unmarshall(dynamodb.NewImage);
      if (newItem.type === 'order') {
        await publishToSNS('order.created', newItem);
      }
    } else if (eventName === 'MODIFY') {
      const old = unmarshall(dynamodb.OldImage);
      const updated = unmarshall(dynamodb.NewImage);
      if (old.status !== updated.status) {
        await publishToSNS('order.status_changed', updated);
      }
    }
  }
};
```

## PartiQL Queries

```javascript
import { ExecuteStatementCommand } from '@aws-sdk/client-dynamodb';

// SELECT
const result = await client.send(new ExecuteStatementCommand({
  Statement: "SELECT * FROM MyApp WHERE PK = ? AND begins_with(SK, ?)",
  Parameters: [{ S: 'USER#alice' }, { S: 'ORDER#' }],
}));
```

## Condition Expressions for Optimistic Locking

```javascript
// Version-based optimistic locking
const updateResult = await ddb.send(new PutCommand({
  TableName: 'MyApp',
  Item: { PK: 'PRODUCT#prod-1', SK: 'METADATA', ...updatedData, version: currentVersion + 1 },
  ConditionExpression: 'attribute_exists(PK) AND #v = :expectedVersion',
  ExpressionAttributeNames: { '#v': 'version' },
  ExpressionAttributeValues: { ':expectedVersion': currentVersion },
}));

// Atomic counter
await ddb.send(new UpdateCommand({
  TableName: 'MyApp',
  Key: { PK: 'COUNTER#views', SK: 'METADATA' },
  UpdateExpression: 'ADD #count :one',
  ExpressionAttributeNames: { '#count': 'count' },
  ExpressionAttributeValues: { ':one': 1 },
}));
```

## TTL Attribute

```javascript
// Add TTL field (Unix epoch seconds)
const expiresAt = Math.floor(Date.now() / 1000) + 7 * 24 * 60 * 60; // 7 days

await ddb.send(new PutCommand({
  TableName: 'MyApp',
  Item: {
    PK: 'SESSION#token123',
    SK: 'METADATA',
    type: 'session',
    userId: 'alice',
    ttl: expiresAt,  // must enable TTL on this attribute in AWS Console / CLI
  },
}));

// Enable TTL on table (CLI):
// aws dynamodb update-time-to-live --table-name MyApp --time-to-live-specification Enabled=true,AttributeName=ttl
```

## Transactions

```javascript
// All-or-nothing across multiple items
await ddb.send(new TransactWriteCommand({
  TransactItems: [
    {
      Update: {
        TableName: 'MyApp',
        Key: { PK: 'PRODUCT#prod-1', SK: 'METADATA' },
        UpdateExpression: 'ADD stock :neg',
        ConditionExpression: 'stock >= :qty',
        ExpressionAttributeValues: { ':neg': -2, ':qty': 2 },
      },
    },
    {
      Put: {
        TableName: 'MyApp',
        Item: { PK: 'USER#alice', SK: 'ORDER#uuid', total: 19.98, status: 'pending' },
        ConditionExpression: 'attribute_not_exists(PK)',
      },
    },
  ],
}));
```

## Key Rules

- Design around access patterns first — there is no ad-hoc query, only planned access patterns
- GSIs have eventual consistency; LSIs are strongly consistent
- Avoid hot partitions: distribute writes across many distinct PK values
- On-demand capacity absorbs spikes; provisioned is cheaper for steady workloads
- Item size limit is 400 KB — store large blobs in S3 and reference by key
- Transactions consume 2x read/write capacity units
- Stream records expire after 24 hours — consume promptly or use Lambda trigger
