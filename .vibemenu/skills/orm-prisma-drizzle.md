# Prisma & Drizzle ORM Skill Guide

---

## Prisma (TypeScript / Node.js)

### Installation

```bash
npm install @prisma/client
npm install -D prisma
npx prisma init
```

### schema.prisma

```prisma
generator client {
  provider = "prisma-client-js"
}

datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

model User {
  id        String    @id @default(cuid())
  email     String    @unique
  name      String?
  role      Role      @default(USER)
  createdAt DateTime  @default(now()) @map("created_at")
  updatedAt DateTime  @updatedAt @map("updated_at")
  posts     Post[]

  @@map("users")
  @@index([email])
}

model Post {
  id        String   @id @default(cuid())
  title     String
  content   String?
  published Boolean  @default(false)
  authorId  String   @map("author_id")
  author    User     @relation(fields: [authorId], references: [id], onDelete: Cascade)
  createdAt DateTime @default(now()) @map("created_at")

  @@map("posts")
  @@index([authorId])
}

enum Role {
  USER
  ADMIN
}
```

### Migration Workflow

```bash
# Development: create and apply migration
npx prisma migrate dev --name add_user_role

# Production: apply pending migrations only (no schema drift check)
npx prisma migrate deploy

# Inspect current DB state
npx prisma migrate status

# Reset DB (dev only — drops and recreates)
npx prisma migrate reset

# Generate client after schema changes (also runs in migrate dev)
npx prisma generate
```

### Client Usage

```typescript
import { PrismaClient } from "@prisma/client";

const prisma = new PrismaClient();

// Create
const user = await prisma.user.create({
  data: { email: "alice@example.com", name: "Alice" },
});

// Find with eager loading
const userWithPosts = await prisma.user.findUnique({
  where: { email: "alice@example.com" },
  include: { posts: { where: { published: true } } },
});

// Projection (select)
const userSlim = await prisma.user.findMany({
  select: { id: true, email: true, name: true },
  where: { role: "ADMIN" },
  orderBy: { createdAt: "desc" },
  take: 20,
  skip: 0,
});

// Update
const updated = await prisma.user.update({
  where: { id: user.id },
  data: { name: "Alice Smith" },
});

// Delete
await prisma.user.delete({ where: { id: user.id } });
```

### Transactions

```typescript
// Sequential transaction
const [newUser, newPost] = await prisma.$transaction([
  prisma.user.create({ data: { email: "bob@example.com" } }),
  prisma.post.create({ data: { title: "Hello", authorId: "some-id" } }),
]);

// Interactive transaction (for conditional logic)
const result = await prisma.$transaction(async (tx) => {
  const user = await tx.user.findUnique({ where: { id: userId } });
  if (!user) throw new Error("User not found");

  return tx.post.create({
    data: { title, content, authorId: user.id },
  });
});
```

### Custom SQL Migration (for CONCURRENTLY, etc.)

```bash
npx prisma migrate dev --create-only --name add_email_index
```

```sql
-- prisma/migrations/20240115_add_email_index/migration.sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email ON users (email);
```

---

## Drizzle ORM (TypeScript / Node.js)

### Installation

```bash
npm install drizzle-orm postgres
npm install -D drizzle-kit
```

### Schema Definition

```typescript
// src/db/schema.ts
import {
  pgTable, text, timestamp, uuid, boolean, integer, pgEnum
} from "drizzle-orm/pg-core";
import { relations } from "drizzle-orm";

export const roleEnum = pgEnum("role", ["USER", "ADMIN"]);

export const users = pgTable("users", {
  id:        uuid("id").primaryKey().defaultRandom(),
  email:     text("email").notNull().unique(),
  name:      text("name"),
  role:      roleEnum("role").notNull().default("USER"),
  createdAt: timestamp("created_at").notNull().defaultNow(),
  updatedAt: timestamp("updated_at").notNull().defaultNow(),
});

export const posts = pgTable("posts", {
  id:        uuid("id").primaryKey().defaultRandom(),
  title:     text("title").notNull(),
  content:   text("content"),
  published: boolean("published").notNull().default(false),
  authorId:  uuid("author_id").notNull().references(() => users.id, { onDelete: "cascade" }),
  createdAt: timestamp("created_at").notNull().defaultNow(),
});

// Relations (for query builder joins)
export const usersRelations = relations(users, ({ many }) => ({
  posts: many(posts),
}));

export const postsRelations = relations(posts, ({ one }) => ({
  author: one(users, { fields: [posts.authorId], references: [users.id] }),
}));
```

### drizzle.config.ts

```typescript
import { defineConfig } from "drizzle-kit";

export default defineConfig({
  schema: "./src/db/schema.ts",
  out:    "./drizzle",
  dialect: "postgresql",
  dbCredentials: { url: process.env.DATABASE_URL! },
});
```

### Migration Workflow

```bash
# Generate migration SQL from schema changes
npx drizzle-kit generate

# Apply migrations to the database
npx drizzle-kit migrate

# Push schema directly to DB (dev only, no migration file generated)
npx drizzle-kit push

# Open Drizzle Studio (GUI)
npx drizzle-kit studio
```

### Client & Queries

```typescript
import { drizzle } from "drizzle-orm/postgres-js";
import postgres from "postgres";
import { eq, and, desc, gt } from "drizzle-orm";
import * as schema from "./schema";

const sql = postgres(process.env.DATABASE_URL!);
const db = drizzle(sql, { schema });

// Insert
const [user] = await db
  .insert(schema.users)
  .values({ email: "alice@example.com", name: "Alice" })
  .returning();

// Select with filter
const admins = await db
  .select({ id: schema.users.id, email: schema.users.email })
  .from(schema.users)
  .where(eq(schema.users.role, "ADMIN"))
  .orderBy(desc(schema.users.createdAt))
  .limit(20);

// Relations query (eager load)
const usersWithPosts = await db.query.users.findMany({
  with: {
    posts: { where: eq(schema.posts.published, true) },
  },
});

// Update
await db
  .update(schema.users)
  .set({ name: "Alice Smith" })
  .where(eq(schema.users.id, user.id));

// Delete
await db.delete(schema.users).where(eq(schema.users.id, user.id));
```

### Transactions

```typescript
await db.transaction(async (tx) => {
  const [newUser] = await tx
    .insert(schema.users)
    .values({ email: "bob@example.com" })
    .returning();

  await tx.insert(schema.posts).values({
    title: "First post",
    authorId: newUser.id,
  });
});
```

## Anti-Patterns

- Prisma: Never edit generated migration files after applying — create a new migration.
- Prisma: Use `$transaction` with interactive function for multi-step logic; array form for atomic batch.
- Drizzle: `push` is for dev only; always use `generate`+`migrate` in production.
- Drizzle: Import `eq`, `and`, `or` operators from `drizzle-orm`, not from the table — they are type-safe wrappers.
