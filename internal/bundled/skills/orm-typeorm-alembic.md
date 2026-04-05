# TypeORM & Alembic Skill Guide

---

## TypeORM (TypeScript / Node.js)

### Installation

```bash
npm install typeorm reflect-metadata pg
npm install -D @types/node
# tsconfig.json: "emitDecoratorMetadata": true, "experimentalDecorators": true
```

### Entity Definition

```typescript
// src/entity/User.ts
import {
  Entity, PrimaryGeneratedColumn, Column, CreateDateColumn,
  UpdateDateColumn, OneToMany, Index,
} from "typeorm";
import { Post } from "./Post";

@Entity("users")
@Index(["email"], { unique: true })
export class User {
  @PrimaryGeneratedColumn("uuid")
  id: string;

  @Column({ length: 255 })
  @Index()
  email: string;

  @Column({ nullable: true })
  name: string | null;

  @Column({ default: "USER" })
  role: string;

  @CreateDateColumn({ name: "created_at" })
  createdAt: Date;

  @UpdateDateColumn({ name: "updated_at" })
  updatedAt: Date;

  @OneToMany(() => Post, (post) => post.author)
  posts: Post[];
}
```

```typescript
// src/entity/Post.ts
import {
  Entity, PrimaryGeneratedColumn, Column, ManyToOne,
  JoinColumn, CreateDateColumn,
} from "typeorm";
import { User } from "./User";

@Entity("posts")
export class Post {
  @PrimaryGeneratedColumn("uuid")
  id: string;

  @Column()
  title: string;

  @Column({ nullable: true, type: "text" })
  content: string | null;

  @Column({ default: false })
  published: boolean;

  @Column({ name: "author_id" })
  authorId: string;

  @ManyToOne(() => User, (user) => user.posts, { onDelete: "CASCADE" })
  @JoinColumn({ name: "author_id" })
  author: User;

  @CreateDateColumn({ name: "created_at" })
  createdAt: Date;
}
```

### Data Source Setup

```typescript
// src/data-source.ts
import { DataSource } from "typeorm";
import { User } from "./entity/User";
import { Post } from "./entity/Post";

export const AppDataSource = new DataSource({
  type: "postgres",
  url: process.env.DATABASE_URL,
  entities: [User, Post],
  migrations: ["src/migration/*.ts"],
  synchronize: false,   // NEVER true in production
  logging: process.env.NODE_ENV === "development",
});
```

### Repository Pattern

```typescript
import { AppDataSource } from "./data-source";
import { User } from "./entity/User";

const userRepo = AppDataSource.getRepository(User);

// Create
const user = userRepo.create({ email: "alice@example.com", name: "Alice" });
await userRepo.save(user);

// Find with eager loading
const userWithPosts = await userRepo.findOne({
  where: { email: "alice@example.com" },
  relations: { posts: true },
});

// Find with select
const admins = await userRepo.find({
  select: { id: true, email: true },
  where: { role: "ADMIN" },
  order: { createdAt: "DESC" },
  take: 20,
});

// Update
await userRepo.update({ id: user.id }, { name: "Alice Smith" });

// Delete
await userRepo.delete({ id: user.id });
```

### QueryBuilder

```typescript
const results = await AppDataSource
  .getRepository(User)
  .createQueryBuilder("user")
  .leftJoinAndSelect("user.posts", "post", "post.published = :pub", { pub: true })
  .where("user.role = :role", { role: "ADMIN" })
  .andWhere("user.createdAt > :date", { date: new Date("2024-01-01") })
  .orderBy("user.createdAt", "DESC")
  .take(20)
  .skip(0)
  .getMany();
```

### Migration Generation

```bash
# Generate migration from entity changes
npx typeorm migration:generate src/migration/AddUserRole -d src/data-source.ts

# Run pending migrations
npx typeorm migration:run -d src/data-source.ts

# Revert last migration
npx typeorm migration:revert -d src/data-source.ts

# Show migration status
npx typeorm migration:show -d src/data-source.ts
```

### Transaction

```typescript
await AppDataSource.transaction(async (manager) => {
  const user = manager.create(User, { email: "bob@example.com" });
  await manager.save(user);

  const post = manager.create(Post, { title: "Hello", authorId: user.id });
  await manager.save(post);
});
```

---

## Alembic (Python / SQLAlchemy)

### Installation

```bash
pip install alembic sqlalchemy psycopg2-binary
alembic init alembic
```

### alembic/env.py

```python
from sqlalchemy import engine_from_config, pool
from alembic import context
import os

# Import your models so autogenerate can detect them
from myapp.models import Base   # SQLAlchemy declarative base

config = context.config
config.set_main_option("sqlalchemy.url", os.environ["DATABASE_URL"])

target_metadata = Base.metadata

def run_migrations_online() -> None:
    connectable = engine_from_config(
        config.get_section(config.config_ini_section, {}),
        prefix="sqlalchemy.",
        poolclass=pool.NullPool,
    )
    with connectable.connect() as connection:
        context.configure(connection=connection, target_metadata=target_metadata)
        with context.begin_transaction():
            context.run_migrations()
```

### SQLAlchemy Model Example

```python
# myapp/models.py
from sqlalchemy import Column, String, Boolean, DateTime, ForeignKey, func
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import DeclarativeBase, relationship
import uuid

class Base(DeclarativeBase):
    pass

class User(Base):
    __tablename__ = "users"

    id         = Column(UUID(as_uuid=True), primary_key=True, default=uuid.uuid4)
    email      = Column(String(255), nullable=False, unique=True)
    name       = Column(String(255))
    role       = Column(String(50), nullable=False, default="USER")
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

    posts = relationship("Post", back_populates="author", cascade="all, delete-orphan")
```

### Migration Workflow

```bash
# Autogenerate migration from model changes
alembic revision --autogenerate -m "add user role column"

# Apply all pending migrations
alembic upgrade head

# Downgrade by 1 step
alembic downgrade -1

# Show migration history
alembic history

# Show current revision
alembic current
```

### Migration File Pattern

```python
# alembic/versions/2024_001_add_user_role.py
from alembic import op
import sqlalchemy as sa

revision = "2024_001"
down_revision = "2023_005"
branch_labels = None
depends_on = None

def upgrade() -> None:
    op.add_column("users", sa.Column("role", sa.String(50), nullable=False, server_default="USER"))
    op.create_index("ix_users_role", "users", ["role"])

def downgrade() -> None:
    op.drop_index("ix_users_role", "users")
    op.drop_column("users", "role")
```

### Custom SQL in Migration

```python
def upgrade() -> None:
    # Raw SQL for operations Alembic can't express (e.g., CONCURRENTLY)
    op.execute("CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_email ON users (email)")
    # Note: CONCURRENTLY cannot run inside a transaction
    # Use execute_if or connection.execute for non-transactional DDL
```

### Data Migration Pattern

```python
from alembic import op
import sqlalchemy as sa

def upgrade() -> None:
    # Bind for data operations
    bind = op.get_bind()
    bind.execute(
        sa.text("UPDATE users SET display_name = username WHERE display_name IS NULL")
    )
```

## Anti-Patterns

- TypeORM: Never set `synchronize: true` in production — it can drop columns.
- TypeORM: Use `getRepository()` not `getConnection()` (deprecated).
- Alembic: Never edit a migration that has already been applied in production — create a new one.
- Alembic: Always import all models in `env.py` before setting `target_metadata` — autogenerate won't detect unimported models.
- Both: Keep DDL (schema changes) and DML (data backfills) in separate migrations.
