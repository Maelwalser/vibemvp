# GraphQL Skill Guide

## Overview

GraphQL is a query language for APIs that gives clients precise control over what data they fetch. Use schema-first (SDL) to define the contract before implementing resolvers.

## Schema Definition Language (SDL)

```graphql
# Scalar types
scalar DateTime
scalar JSON

# Object type
type User {
  id: ID!
  email: String!
  name: String!
  posts: [Post!]!
  createdAt: DateTime!
}

type Post {
  id: ID!
  title: String!
  body: String!
  author: User!
  published: Boolean!
}

# Query root
type Query {
  user(id: ID!): User
  users(limit: Int = 20, offset: Int = 0): [User!]!
  post(id: ID!): Post
}

# Mutation root
type Mutation {
  createUser(input: CreateUserInput!): UserPayload!
  updatePost(id: ID!, input: UpdatePostInput!): PostPayload!
  deletePost(id: ID!): Boolean!
}

# Subscription root
type Subscription {
  postPublished(authorId: ID): Post!
  commentAdded(postId: ID!): Comment!
}

# Input types
input CreateUserInput {
  email: String!
  name: String!
  password: String!
}

input UpdatePostInput {
  title: String
  body: String
  published: Boolean
}

# Payload types (mutation results with errors)
type UserPayload {
  user: User
  errors: [UserError!]!
}

type UserError {
  field: String!
  message: String!
}

# Enums
enum PostStatus {
  DRAFT
  PUBLISHED
  ARCHIVED
}

# Interfaces
interface Node {
  id: ID!
}

type Product implements Node {
  id: ID!
  name: String!
}

# Unions
union SearchResult = User | Post | Product
```

## Resolvers with Context and Args

### Node.js (Apollo Server / GraphQL Yoga)

```typescript
import { GraphQLResolveInfo } from "graphql";

interface Context {
  user: { id: string; role: string } | null;
  db: Database;
  loaders: DataLoaders;
}

const resolvers = {
  Query: {
    user: async (
      _parent: unknown,
      args: { id: string },
      ctx: Context,
      _info: GraphQLResolveInfo,
    ) => {
      return ctx.db.users.findById(args.id);
    },

    users: async (
      _parent: unknown,
      args: { limit: number; offset: number },
      ctx: Context,
    ) => {
      return ctx.db.users.findAll({ limit: args.limit, offset: args.offset });
    },
  },

  Mutation: {
    createUser: async (_parent: unknown, { input }: { input: CreateUserInput }, ctx: Context) => {
      if (!ctx.user) throw new GraphQLError("Unauthorized", { extensions: { code: "UNAUTHORIZED" } });
      try {
        const user = await ctx.db.users.create(input);
        return { user, errors: [] };
      } catch (err) {
        return { user: null, errors: [{ field: "email", message: "Email already taken" }] };
      }
    },
  },

  // Field resolvers — use DataLoader to batch
  User: {
    posts: (parent: User, _args: unknown, ctx: Context) => {
      return ctx.loaders.postsByAuthor.load(parent.id);
    },
  },

  Post: {
    author: (parent: Post, _args: unknown, ctx: Context) => {
      return ctx.loaders.userById.load(parent.authorId);
    },
  },

  // Union/Interface resolver
  SearchResult: {
    __resolveType(obj: unknown) {
      if ("email" in (obj as object)) return "User";
      if ("title" in (obj as object)) return "Post";
      return "Product";
    },
  },
};
```

### Python (Strawberry)

```python
import strawberry
from strawberry.types import Info
from typing import Optional

@strawberry.type
class User:
    id: strawberry.ID
    email: str
    name: str

@strawberry.type
class Query:
    @strawberry.field
    async def user(self, info: Info, id: strawberry.ID) -> Optional[User]:
        return await info.context["db"].users.find_by_id(id)

@strawberry.type
class Mutation:
    @strawberry.mutation
    async def create_user(self, info: Info, email: str, name: str) -> User:
        return await info.context["db"].users.create(email=email, name=name)

schema = strawberry.Schema(query=Query, mutation=Mutation)
```

### Go (gqlgen)

```go
// resolver.go
func (r *queryResolver) User(ctx context.Context, id string) (*model.User, error) {
    return r.db.Users.FindByID(ctx, id)
}

func (r *mutationResolver) CreateUser(ctx context.Context, input model.CreateUserInput) (*model.UserPayload, error) {
    user, err := r.db.Users.Create(ctx, input)
    if err != nil {
        return &model.UserPayload{Errors: []model.UserError{{Field: "email", Message: err.Error()}}}, nil
    }
    return &model.UserPayload{User: user}, nil
}
```

## DataLoader Batching (N+1 Prevention)

The N+1 problem: fetching a list of posts and then individually fetching each author triggers N+1 queries. DataLoader batches these into a single query per tick.

```typescript
import DataLoader from "dataloader";

// Create loaders per request (not shared across requests)
function createLoaders(db: Database) {
  return {
    userById: new DataLoader<string, User>(async (ids) => {
      // Single batch query for all ids
      const users = await db.users.findByIds([...ids]);
      const userMap = new Map(users.map(u => [u.id, u]));
      // Must return results in same order as input ids
      return ids.map(id => userMap.get(id) ?? new Error(`User ${id} not found`));
    }),

    postsByAuthor: new DataLoader<string, Post[]>(async (authorIds) => {
      const posts = await db.posts.findByAuthorIds([...authorIds]);
      const postsByAuthor = new Map<string, Post[]>();
      for (const post of posts) {
        const list = postsByAuthor.get(post.authorId) ?? [];
        list.push(post);
        postsByAuthor.set(post.authorId, list);
      }
      return authorIds.map(id => postsByAuthor.get(id) ?? []);
    }),
  };
}

// Attach to context in server setup
const server = new ApolloServer({
  typeDefs,
  resolvers,
  context: ({ req }) => ({
    user: getUser(req),
    db,
    loaders: createLoaders(db), // fresh per request
  }),
});
```

## Subscriptions with PubSub

```typescript
import { PubSub } from "graphql-subscriptions";

const pubsub = new PubSub();
const POST_PUBLISHED = "POST_PUBLISHED";

const resolvers = {
  Mutation: {
    publishPost: async (_parent: unknown, { id }: { id: string }, ctx: Context) => {
      const post = await ctx.db.posts.publish(id);
      // Publish event after successful mutation
      await pubsub.publish(POST_PUBLISHED, { postPublished: post });
      return post;
    },
  },

  Subscription: {
    postPublished: {
      subscribe: (_parent: unknown, { authorId }: { authorId?: string }) => {
        const asyncIterator = pubsub.asyncIterator([POST_PUBLISHED]);
        if (!authorId) return asyncIterator;
        // Filter by authorId if provided
        return {
          [Symbol.asyncIterator]() {
            return {
              async next() {
                while (true) {
                  const result = await asyncIterator[Symbol.asyncIterator]().next();
                  if (result.done || result.value.postPublished.authorId === authorId) {
                    return result;
                  }
                }
              },
            };
          },
        };
      },
      resolve: (payload: { postPublished: Post }) => payload.postPublished,
    },
  },
};
```

## Schema-First vs Code-First

| Approach | Tooling | Best For |
|----------|---------|---------|
| Schema-first (SDL) | graphql-tools, gqlgen | API-design-first teams, shared SDL |
| Code-first | type-graphql (TS), Strawberry (Python), gqlgen (Go gen) | Type safety, single source of truth |

### Code-First with type-graphql

```typescript
import { ObjectType, Field, ID, Resolver, Query, Arg, Ctx } from "type-graphql";

@ObjectType()
class User {
  @Field(() => ID)
  id: string;

  @Field()
  email: string;

  @Field()
  name: string;
}

@Resolver(User)
class UserResolver {
  @Query(() => User, { nullable: true })
  async user(@Arg("id") id: string, @Ctx() ctx: Context): Promise<User | null> {
    return ctx.db.users.findById(id);
  }
}
```

## Depth Limiting

```typescript
import depthLimit from "graphql-depth-limit";

const server = new ApolloServer({
  typeDefs,
  resolvers,
  validationRules: [depthLimit(5)], // reject queries deeper than 5 levels
});
```

## Persisted Queries

```typescript
import { ApolloServerPluginCacheControl } from "@apollo/server/plugin/cacheControl";
import { createHash } from "crypto";

// Client sends SHA-256 hash of query instead of full query
// Server caches hash → query mapping to save bandwidth and enable allowlisting

const server = new ApolloServer({
  typeDefs,
  resolvers,
  plugins: [ApolloServerPluginCacheControl()],
  // With Automatic Persisted Queries (APQ):
  // 1. Client sends { extensions: { persistedQuery: { version: 1, sha256Hash: "abc..." } } }
  // 2. If miss: server returns PersistedQueryNotFound
  // 3. Client resends with full query + hash
  // 4. Server caches and responds
});
```

## Rules

- Always create DataLoaders per request — never share across requests (stale data)
- Use input types for all mutation arguments — never use inline scalars for complex inputs
- Return payload types from mutations (`{ data, errors[] }`) instead of throwing for user errors
- Validate query depth and complexity to prevent abuse
- Use `extensions.code` on GraphQLError for machine-readable error codes
- Field resolvers default to returning `parent[fieldName]` — only write them when you need transformation or batching
- Never expose internal error details in production — use generic messages with error codes
