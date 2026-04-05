# Redis & Memcached Skill Guide

## Overview

Redis is an in-memory data structure store supporting rich data types, persistence, pub/sub, and scripting. Memcached is a simpler high-performance distributed cache focused purely on key-value storage.

## Redis Setup & Connection

```javascript
// ioredis (Node.js — recommended)
import Redis from 'ioredis';

const redis = new Redis({
  host: process.env.REDIS_HOST,
  port: 6379,
  password: process.env.REDIS_PASSWORD,
  tls: process.env.REDIS_TLS === 'true' ? {} : undefined,
  maxRetriesPerRequest: 3,
  enableReadyCheck: true,
  retryStrategy: (times) => Math.min(times * 50, 2000),
});

// Cluster mode
const cluster = new Redis.Cluster([
  { host: 'redis-1', port: 6379 },
  { host: 'redis-2', port: 6379 },
]);
```

```go
// Go — redis/v9
import "github.com/redis/go-redis/v9"

rdb := redis.NewClient(&redis.Options{
    Addr:     os.Getenv("REDIS_ADDR"),
    Password: os.Getenv("REDIS_PASSWORD"),
    DB:       0,
    PoolSize: 10,
})
```

## Data Types

```javascript
// STRING — cache, counters, sessions
await redis.set('user:alice:session', JSON.stringify(session), 'EX', 3600);
await redis.get('user:alice:session');
await redis.incr('page:views');            // atomic counter
await redis.incrby('score:alice', 10);

// HASH — structured object without serialization overhead
await redis.hset('user:alice', { name: 'Alice', email: 'alice@example.com', role: 'admin' });
await redis.hget('user:alice', 'name');
await redis.hgetall('user:alice');
await redis.hincrby('user:alice', 'loginCount', 1);

// LIST — queues, timelines (LPUSH + BRPOP = reliable queue)
await redis.lpush('queue:emails', JSON.stringify(job));
const item = await redis.brpop('queue:emails', 5);  // block up to 5s

// SET — unique membership, tags, permissions
await redis.sadd('online:users', 'alice', 'bob');
await redis.sismember('online:users', 'alice');
await redis.sinter('group:admins', 'online:users');  // intersection

// SORTED SET — leaderboards, rate limiting, delayed queues
await redis.zadd('leaderboard', 1500, 'alice', 850, 'bob');
await redis.zrevrange('leaderboard', 0, 9, 'WITHSCORES');  // top 10
await redis.zincrby('leaderboard', 100, 'alice');

// STREAM — event log with consumer groups
await redis.xadd('events', '*', 'type', 'order.created', 'userId', 'alice');
await redis.xreadgroup('GROUP', 'workers', 'worker-1', 'COUNT', 10, 'STREAMS', 'events', '>');
```

## TTL Management

```javascript
// Set TTL on creation
await redis.set('token:abc', userId, 'EX', 900);       // 900 seconds
await redis.set('otp:alice', '123456', 'PX', 300000);  // 300,000 ms

// Set TTL on existing key
await redis.expire('session:xyz', 1800);
await redis.expireat('deal:flash', Math.floor(new Date('2024-12-31').getTime() / 1000));

// Check remaining TTL
const ttl = await redis.ttl('session:xyz');  // -1 = no expiry, -2 = key doesn't exist

// Refresh TTL on access (sliding expiration)
await redis.getex('session:xyz', 'EX', 1800);
```

## Eviction Policies

```
Policy               Behavior
─────────────────────────────────────────────────────────────────
noeviction           Return error when max memory reached (default)
allkeys-lru          Evict LRU key from all keys (best for general cache)
volatile-lru         Evict LRU key from keys with TTL set
allkeys-lfu          Evict least-frequently-used key (LFU — Redis 4+)
volatile-lfu         Evict LFU key from keys with TTL set
allkeys-random       Evict random key from all keys
volatile-random      Evict random key from keys with TTL set
volatile-ttl         Evict key with shortest remaining TTL
```

```bash
# Configure in redis.conf or via CONFIG SET
redis-cli CONFIG SET maxmemory 2gb
redis-cli CONFIG SET maxmemory-policy allkeys-lru
```

## Lua Scripting for Atomic Operations

```javascript
// Atomic rate limiter using Lua (token bucket)
const rateLimitScript = `
  local key = KEYS[1]
  local limit = tonumber(ARGV[1])
  local window = tonumber(ARGV[2])
  local current = redis.call('INCR', key)
  if current == 1 then
    redis.call('EXPIRE', key, window)
  end
  if current > limit then
    return 0
  end
  return 1
`;

const allowed = await redis.eval(rateLimitScript, 1, `ratelimit:${userId}`, 100, 60);
if (!allowed) throw new Error('Rate limit exceeded');
```

## Pub/Sub

```javascript
// Publisher
const publisher = new Redis(redisConfig);
await publisher.publish('notifications', JSON.stringify({ userId: 'alice', message: 'Hello' }));

// Subscriber (dedicated connection — cannot run other commands)
const subscriber = new Redis(redisConfig);
await subscriber.subscribe('notifications');
subscriber.on('message', (channel, message) => {
  const data = JSON.parse(message);
  broadcastToUser(data.userId, data.message);
});

// Pattern subscribe
await subscriber.psubscribe('notifications:*');
subscriber.on('pmessage', (pattern, channel, message) => { /* ... */ });
```

## Persistence (RDB vs AOF)

```bash
# RDB — periodic snapshot (faster restart, potential data loss)
save 900 1       # save if 1 key changed in 900 seconds
save 300 10      # save if 10 keys changed in 300 seconds
save 60 10000    # save if 10000 keys changed in 60 seconds

# AOF — append-only file (better durability)
appendonly yes
appendfsync everysec   # fsync every second (good balance)
# appendfsync always   # fsync every write (slowest, safest)
# appendfsync no       # let OS decide (fastest, least safe)

# Hybrid (recommended for production)
aof-use-rdb-preamble yes
```

## Memcached Setup & Connection

```javascript
import Memcached from 'memcached';

const memcached = new Memcached(['cache-1:11211', 'cache-2:11211'], {
  retries: 3,
  timeout: 500,
  poolSize: 10,
  consistent: true,   // consistent hashing for stable key distribution
});

// Set, Get, Delete
memcached.set('key', JSON.stringify(value), 3600, (err) => { if (err) throw err; });
memcached.get('key', (err, data) => { if (!err) return JSON.parse(data); });
memcached.del('key', (err) => { /* ... */ });

// Multi-get (batch fetch — highly efficient)
memcached.getMulti(['key1', 'key2', 'key3'], (err, results) => {
  // results = { key1: ..., key2: ... }
});
```

## Memcached: Slab Allocator

Memcached divides memory into slabs (size classes). Items are stored in the smallest slab that fits.

```bash
# View slab stats
echo "stats slabs" | nc cache-host 11211

# Configure slab growth factor (default 1.25)
memcached -f 1.25 -m 512  # 512 MB max memory

# Slab classes: 64B, 80B, 104B, 136B, ... up to 1MB
# Store items consistently sized to maximize slab utilization
```

## Key Rules

- Use separate Redis connections for pub/sub and command execution
- Never use `KEYS *` in production — use `SCAN` with cursor instead
- Pipeline commands to reduce round-trips: `const pipeline = redis.pipeline(); pipeline.get(k1); pipeline.set(k2, v); await pipeline.exec();`
- Redis is single-threaded per shard — avoid long-running Lua scripts (blocks all commands)
- Memcached has no persistence — do not store anything you cannot reconstruct
- Consistent hashing in Memcached means adding/removing a server only remaps ~1/N keys
- Set `maxmemory` and an eviction policy in production — without it, Redis will OOM
