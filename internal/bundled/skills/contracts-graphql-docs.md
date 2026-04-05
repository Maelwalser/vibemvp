# Contracts: GraphQL Documentation Skill Guide

## Overview

GraphQL SDL type definitions, Apollo Sandbox, schema introspection, GraphQL Playground, and schema-first vs code-first generation.

## GraphQL SDL

### Type Definitions

```graphql
# schema.graphql

scalar DateTime
scalar UUID

# Interfaces
interface Node {
  id: ID!
}

interface Timestamps {
  createdAt: DateTime!
  updatedAt: DateTime!
}

# Enums
enum Role {
  USER
  ADMIN
  MODERATOR
}

enum OrderStatus {
  PENDING
  CONFIRMED
  SHIPPED
  DELIVERED
  CANCELLED
}

# Object types
type User implements Node & Timestamps {
  id: ID!
  email: String!
  name: String!
  role: Role!
  orders(first: Int, after: String): OrderConnection!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type Order implements Node {
  id: ID!
  status: OrderStatus!
  total: Float!
  items: [OrderItem!]!
  user: User!
  createdAt: DateTime!
}

type OrderItem {
  product: Product!
  quantity: Int!
  unitPrice: Float!
}

# Pagination (Relay-style)
type OrderConnection {
  edges: [OrderEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type OrderEdge {
  node: Order!
  cursor: String!
}

type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

# Union types
union SearchResult = User | Order | Product

# Input types for mutations
input CreateUserInput {
  email: String!
  name: String!
  role: Role = USER
}

input UpdateUserInput {
  name: String
  role: Role
}

# Root types
type Query {
  # User queries
  user(id: ID!): User
  users(first: Int, after: String, role: Role): UserConnection!
  me: User

  # Search
  search(query: String!): [SearchResult!]!
}

type Mutation {
  # User mutations
  createUser(input: CreateUserInput!): User!
  updateUser(id: ID!, input: UpdateUserInput!): User!
  deleteUser(id: ID!): Boolean!

  # Auth
  login(email: String!, password: String!): AuthPayload!
  logout: Boolean!
}

type Subscription {
  # Real-time events
  orderStatusChanged(orderId: ID!): Order!
  userCreated: User!
}

type AuthPayload {
  token: String!
  user: User!
}
```

## Apollo Server Setup (TypeScript)

```typescript
import { ApolloServer } from '@apollo/server';
import { expressMiddleware } from '@apollo/server/express4';
import { ApolloServerPluginDrainHttpServer } from '@apollo/server/plugin/drainHttpServer';
import { readFileSync } from 'fs';

const typeDefs = readFileSync('./schema.graphql', 'utf-8');

const server = new ApolloServer({
  typeDefs,
  resolvers,
  plugins: [
    ApolloServerPluginDrainHttpServer({ httpServer }),
  ],
  // Apollo Sandbox only in development
  introspection: process.env.NODE_ENV !== 'production',
});

await server.start();

app.use(
  '/graphql',
  cors(),
  express.json(),
  expressMiddleware(server, {
    context: async ({ req }) => ({
      user: await authenticate(req.headers.authorization),
      dataSources: { userAPI: new UserAPI(), orderAPI: new OrderAPI() },
    }),
  }),
);
```

## Apollo Sandbox

```typescript
// Enable Apollo Sandbox in development (replaces Playground)
// Automatically available at /graphql when introspection is enabled

// Force enable introspection in staging (not production)
const server = new ApolloServer({
  typeDefs,
  resolvers,
  introspection: process.env.ENABLE_INTROSPECTION === 'true',
});
```

```bash
# Access Apollo Sandbox
# Open https://studio.apollographql.com/sandbox in browser
# Point it to http://localhost:4000/graphql
```

## GraphQL Playground (dev only)

```typescript
// graphql-playground-middleware-express
import expressPlayground from 'graphql-playground-middleware-express';

if (process.env.NODE_ENV !== 'production') {
  app.get('/playground', expressPlayground({ endpoint: '/graphql' }));
}
```

## Schema-First vs Code-First

### Schema-First (write SDL → codegen generates types)

```bash
# graphql-codegen.yml
overwrite: true
schema: ./schema.graphql
generates:
  src/generated/graphql.ts:
    plugins:
      - typescript
      - typescript-resolvers
    config:
      useIndexSignature: true
      mappers:
        User: '../models#UserModel'
        Order: '../models#OrderModel'
```

```bash
# Generate types
npx graphql-codegen
```

### Code-First (TypeScript — type-graphql)

```typescript
import { ObjectType, Field, ID, registerEnumType } from 'type-graphql';

registerEnumType(Role, { name: 'Role' });

@ObjectType()
class User implements Node {
  @Field(() => ID)
  id: string;

  @Field()
  email: string;

  @Field()
  name: string;

  @Field(() => Role)
  role: Role;

  @Field()
  createdAt: Date;
}

@Resolver(() => User)
class UserResolver {
  @Query(() => User, { nullable: true })
  async user(@Arg('id', () => ID) id: string): Promise<User | null> {
    return userService.findById(id);
  }

  @Mutation(() => User)
  async createUser(@Arg('input') input: CreateUserInput): Promise<User> {
    return userService.create(input);
  }

  @Subscription(() => User)
  userCreated(): AsyncIterator<User> {
    return pubSub.asyncIterator('USER_CREATED');
  }
}
```

### Code-First (Python — strawberry)

```python
import strawberry
from typing import Optional
from datetime import datetime

@strawberry.type
class User:
    id: strawberry.ID
    email: str
    name: str
    created_at: datetime

@strawberry.input
class CreateUserInput:
    email: str
    name: str

@strawberry.type
class Query:
    @strawberry.field
    async def user(self, id: strawberry.ID) -> Optional[User]:
        return await user_service.find_by_id(id)

@strawberry.type
class Mutation:
    @strawberry.mutation
    async def create_user(self, input: CreateUserInput) -> User:
        return await user_service.create(input)

schema = strawberry.Schema(query=Query, mutation=Mutation)
app = strawberry.FastAPI(schema=schema)
```

### Code-First (Go — gqlgen)

```yaml
# gqlgen.yml
schema:
  - graph/*.graphqls
exec:
  filename: graph/generated.go
model:
  filename: graph/model/models_gen.go
resolver:
  layout: follow-schema
  dir: graph
  package: graph
  filename_template: "{name}.resolvers.go"
```

```bash
# Generate boilerplate from schema
go run github.com/99designs/gqlgen generate
```

## Key Rules

- Keep SDL as the source of truth — even in code-first projects, generate the SDL and commit it.
- Disable introspection in production — it exposes your full API surface to attackers.
- Enable Apollo Sandbox only in development and staging environments.
- Use Relay-style connection/edge/pageInfo for all paginated lists — it's the standard.
- Always use `input` types for mutation arguments — never pass raw scalars as multiple args.
- Use `!` (non-null) aggressively — nullable fields should represent genuinely optional data.
- Add descriptions to all types, fields, and arguments — they appear as docs in Sandbox/Playground.
- Run `graphql-inspector` in CI to detect breaking schema changes.
