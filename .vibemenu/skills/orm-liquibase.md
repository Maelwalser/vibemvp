# Liquibase Skill Guide

## Installation

```bash
# Homebrew
brew install liquibase

# Docker
docker pull liquibase/liquibase:latest

# Maven dependency
# <dependency>
#   <groupId>org.liquibase</groupId>
#   <artifactId>liquibase-core</artifactId>
# </dependency>
```

## Changelog Formats

Liquibase supports XML, YAML, JSON, and SQL changelog formats.

### XML (most explicit)

```xml
<!-- db/changelog/db.changelog-master.xml -->
<?xml version="1.0" encoding="UTF-8"?>
<databaseChangeLog
    xmlns="http://www.liquibase.org/xml/ns/dbchangelog"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xsi:schemaLocation="http://www.liquibase.org/xml/ns/dbchangelog
        http://www.liquibase.org/xml/ns/dbchangelog/dbchangelog-latest.xsd">

    <!-- Include child changelogs in order -->
    <include file="db/changelog/001-create-users.xml"/>
    <include file="db/changelog/002-add-user-role.xml"/>
    <include file="db/changelog/003-create-posts.xml"/>
</databaseChangeLog>
```

```xml
<!-- db/changelog/001-create-users.xml -->
<databaseChangeLog ...>
    <changeSet id="001" author="alice">
        <createTable tableName="users">
            <column name="id" type="UUID" defaultValueComputed="gen_random_uuid()">
                <constraints primaryKey="true" nullable="false"/>
            </column>
            <column name="email" type="VARCHAR(255)">
                <constraints nullable="false" unique="true"/>
            </column>
            <column name="name" type="VARCHAR(255)"/>
            <column name="created_at" type="TIMESTAMPTZ" defaultValueComputed="now()">
                <constraints nullable="false"/>
            </column>
        </createTable>
        <createIndex tableName="users" indexName="idx_users_email" unique="true">
            <column name="email"/>
        </createIndex>

        <rollback>
            <dropTable tableName="users"/>
        </rollback>
    </changeSet>
</databaseChangeLog>
```

### YAML Format

```yaml
# db/changelog/002-add-user-role.yaml
databaseChangeLog:
  - changeSet:
      id: "002"
      author: alice
      changes:
        - addColumn:
            tableName: users
            columns:
              - column:
                  name: role
                  type: VARCHAR(50)
                  defaultValue: USER
                  constraints:
                    nullable: false
      rollback:
        - dropColumn:
            tableName: users
            columnName: role
```

### SQL Format

```sql
-- db/changelog/003-create-posts.sql
--liquibase formatted sql

--changeset alice:003
CREATE TABLE posts (
    id         UUID        NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
    title      VARCHAR(500) NOT NULL,
    content    TEXT,
    published  BOOLEAN     NOT NULL DEFAULT false,
    author_id  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_posts_author ON posts (author_id);

--rollback DROP TABLE posts;
```

## changeset Attributes

| Attribute | Description |
|-----------|-------------|
| `id` | Unique within the changelog file |
| `author` | Who wrote the changeset |
| `runOnChange` | Re-run if content changes (like repeatable migrations) |
| `runAlways` | Run every time Liquibase executes |
| `context` | Comma-separated execution contexts (dev, test, prod) |
| `labels` | For filtering, similar to contexts |
| `failOnError` | Whether to halt on error (default true) |

## Preconditions

```xml
<changeSet id="004" author="alice">
    <preConditions onFail="MARK_RAN" onError="WARN">
        <!-- Only run if column doesn't already exist -->
        <not>
            <columnExists tableName="users" columnName="avatar_url"/>
        </not>
    </preConditions>
    <addColumn tableName="users">
        <column name="avatar_url" type="TEXT"/>
    </addColumn>
</changeSet>
```

`onFail` / `onError` values: `HALT` (default), `CONTINUE`, `MARK_RAN`, `WARN`

## Contexts for Environment-Specific Execution

```xml
<!-- Only runs in 'dev' and 'test' contexts -->
<changeSet id="005" author="alice" context="dev,test">
    <sql>INSERT INTO users (email, role) VALUES ('admin@example.com', 'ADMIN')</sql>
    <rollback>DELETE FROM users WHERE email = 'admin@example.com'</rollback>
</changeSet>
```

```bash
# Run with specific context
liquibase --contexts=prod update
liquibase --contexts="dev,test" update
```

## Spring Boot Configuration

```yaml
# application.yml
spring:
  datasource:
    url: ${DATABASE_URL}
    username: ${DB_USER}
    password: ${DB_PASSWORD}
  liquibase:
    change-log: classpath:db/changelog/db.changelog-master.xml
    enabled: true
    contexts: ${SPRING_PROFILES_ACTIVE:dev}
    default-schema: public
    drop-first: false         # NEVER true in production
    clear-checksums: false    # NEVER true in production (use repair instead)
```

Spring Boot auto-applies pending changesets on startup.

## CLI Commands

```bash
# liquibase.properties
url=jdbc:postgresql://localhost/mydb
username=postgres
password=secret
changeLogFile=db/changelog/db.changelog-master.xml
driver=org.postgresql.Driver
```

```bash
liquibase update                          # Apply all pending changesets
liquibase updateCount 3                   # Apply exactly 3 changesets
liquibase rollbackCount 1                 # Rollback 1 changeset
liquibase rollbackToDate 2024-01-15       # Rollback to specific date
liquibase rollbackToTag v1.0              # Rollback to tagged version
liquibase status                          # Show pending changesets
liquibase history                         # Show applied changesets
liquibase validate                        # Validate changelog syntax
liquibase generateChangeLog               # Reverse-engineer from existing DB
liquibase diff                            # Diff two databases
liquibase tag v1.0                        # Tag current state for rollback target
```

## Rollback Strategies

```bash
# Rollback last N changesets
liquibase rollbackCount 3

# Rollback to a date
liquibase rollbackToDate 2024-01-15T12:00:00

# Rollback to a tag
liquibase tag v2.0   # Tag before deployment
# ... later if needed:
liquibase rollbackToTag v2.0
```

Custom rollback in changeset:
```xml
<changeSet id="006" author="alice">
    <addColumn tableName="users">
        <column name="phone" type="VARCHAR(20)"/>
    </addColumn>
    <!-- Without explicit rollback, Liquibase auto-generates DROP COLUMN -->
    <!-- Provide explicit rollback for non-reversible or complex ops: -->
    <rollback>
        <dropColumn tableName="users" columnName="phone"/>
    </rollback>
</changeSet>
```

## Checksum Repair

If a changeset file was accidentally modified after being applied:

```bash
# Reset all checksums (recomputes from files)
liquibase clearChecksums

# Or in Spring Boot (one-time only — reset to false after)
spring.liquibase.clear-checksums: true
```

Better: never modify applied changesets. Create a new one.

## Anti-Patterns

- Never reorder changesets that have already been applied — checksums will fail.
- Never delete a changeset from the changelog once applied — Liquibase will detect missing entries.
- Never modify the `id`/`author` combination of applied changesets.
- Do not use `runAlways: true` on schema DDL — use only for views/functions that need recompiling.
- Separate seed data (contexts: dev) from schema migrations — never mix in prod-targeted changesets.
