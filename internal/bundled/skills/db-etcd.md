# etcd Skill Guide

## Overview

etcd is a distributed, strongly-consistent key-value store used for configuration management, service discovery, and distributed coordination. It uses the Raft consensus algorithm and provides MVCC with watch semantics.

## Setup & Connection

```go
import (
    "context"
    "time"
    clientv3 "go.etcd.io/etcd/client/v3"
    "go.etcd.io/etcd/client/v3/concurrency"
)

func newEtcdClient() (*clientv3.Client, error) {
    return clientv3.New(clientv3.Config{
        Endpoints:   []string{
            "etcd-0:2379",
            "etcd-1:2379",
            "etcd-2:2379",
        },
        DialTimeout: 5 * time.Second,
        TLS:         tlsConfig,  // nil for non-TLS
    })
}
```

## Key-Value CRUD

```go
ctx := context.Background()

// PUT
_, err := client.Put(ctx, "/config/db/host", "postgres:5432")

// PUT with TTL (requires lease)
lease, err := client.Grant(ctx, 30)  // 30 second TTL
_, err = client.Put(ctx, "/locks/resource-1", "worker-1",
    clientv3.WithLease(lease.ID))

// GET single key
resp, err := client.Get(ctx, "/config/db/host")
if len(resp.Kvs) > 0 {
    fmt.Printf("value=%s revision=%d\n", resp.Kvs[0].Value, resp.Kvs[0].ModRevision)
}

// GET prefix (all keys under /config/)
resp, err := client.Get(ctx, "/config/", clientv3.WithPrefix())
for _, kv := range resp.Kvs {
    fmt.Printf("%s = %s\n", kv.Key, kv.Value)
}

// DELETE
_, err = client.Delete(ctx, "/config/db/host")

// DELETE all keys under prefix
_, err = client.Delete(ctx, "/config/", clientv3.WithPrefix())
```

## Watch for Change Notifications

```go
// Watch a single key
watchChan := client.Watch(ctx, "/config/db/host")
go func() {
    for watchResp := range watchChan {
        for _, event := range watchResp.Events {
            switch event.Type {
            case clientv3.EventTypePut:
                fmt.Printf("updated: %s = %s\n", event.Kv.Key, event.Kv.Value)
                reloadConfig(string(event.Kv.Value))
            case clientv3.EventTypeDelete:
                fmt.Printf("deleted: %s\n", event.Kv.Key)
            }
        }
    }
}()

// Watch prefix — receive all changes under /services/
watchChan := client.Watch(ctx, "/services/", clientv3.WithPrefix())

// Resume watch from specific revision (after reconnect)
watchChan := client.Watch(ctx, "/config/", 
    clientv3.WithPrefix(), 
    clientv3.WithRev(lastSeenRevision))
```

## Leases with TTL for Service Registration

```go
// Register service with TTL-based lease
func registerService(client *clientv3.Client, serviceKey, serviceAddr string) error {
    ctx := context.Background()

    // Create a 10-second lease
    lease, err := client.Grant(ctx, 10)
    if err != nil {
        return fmt.Errorf("grant lease: %w", err)
    }

    // Register the service
    _, err = client.Put(ctx, serviceKey, serviceAddr, clientv3.WithLease(lease.ID))
    if err != nil {
        return fmt.Errorf("register service: %w", err)
    }

    // Keep-alive: automatically renew before expiry
    keepAliveChan, err := client.KeepAlive(ctx, lease.ID)
    if err != nil {
        return fmt.Errorf("keepalive: %w", err)
    }

    go func() {
        for range keepAliveChan {
            // drain — KeepAlive renews automatically
        }
        // channel closed = lease expired or ctx cancelled
        log.Println("service lease expired, re-registering")
    }()

    return nil
}
```

## MVCC with Revision Numbers

```go
// Every write increments the cluster revision
// Each key tracks: CreateRevision, ModRevision, Version (per-key write count)

resp, err := client.Get(ctx, "/config/threshold")
kv := resp.Kvs[0]
fmt.Printf("CreateRevision=%d ModRevision=%d Version=%d\n",
    kv.CreateRevision, kv.ModRevision, kv.Version)

// Get key at a specific historical revision
resp, err := client.Get(ctx, "/config/threshold",
    clientv3.WithRev(specificRevision))

// Header.Revision = current cluster revision at time of response
currentRevision := resp.Header.Revision
```

## Transactions (If / Then / Else)

```go
// Compare-and-swap — only update if current value matches
txResp, err := client.Txn(ctx).
    If(clientv3.Compare(clientv3.Value("/locks/leader"), "=", "")).
    Then(clientv3.OpPut("/locks/leader", "node-1")).
    Else(clientv3.OpGet("/locks/leader")).
    Commit()

if txResp.Succeeded {
    fmt.Println("became leader")
} else {
    // Read current leader from Else response
    leader := txResp.Responses[0].GetResponseRange().Kvs[0].Value
    fmt.Printf("current leader: %s\n", leader)
}

// Check key does not exist before creating
txResp, err := client.Txn(ctx).
    If(clientv3.Compare(clientv3.CreateRevision("/config/init"), "=", 0)).
    Then(clientv3.OpPut("/config/init", "done")).
    Commit()
```

## Compaction

```go
// Compact removes old revisions — only keep history back to revision N
// Run periodically to prevent disk growth
_, err := client.Compact(ctx, resp.Header.Revision,
    clientv3.WithCompactPhysical())  // blocks until physically compacted

// Recommended: compact periodically in a background goroutine
func periodicCompaction(client *clientv3.Client) {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        resp, _ := client.Get(context.Background(), "/", clientv3.WithLastRev())
        client.Compact(context.Background(), resp.Header.Revision)
    }
}
```

## Distributed Lock with concurrency.NewMutex

```go
// Distributed mutex using etcd sessions
func withLock(client *clientv3.Client, lockName string, fn func() error) error {
    session, err := concurrency.NewSession(client, concurrency.WithTTL(30))
    if err != nil {
        return fmt.Errorf("create session: %w", err)
    }
    defer session.Close()

    mutex := concurrency.NewMutex(session, "/locks/"+lockName)

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := mutex.Lock(ctx); err != nil {
        return fmt.Errorf("acquire lock: %w", err)
    }
    defer mutex.Unlock(context.Background())

    return fn()
}

// Usage
err := withLock(client, "migrations", func() error {
    return runMigrations()
})
```

## Key Rules

- etcd is optimized for small values (config, metadata) — do not store blobs or large datasets
- Maximum recommended value size is 1.5 MB; cluster recommended for values up to 8 MB
- Always set `DialTimeout` — etcd operations can block indefinitely without it
- Use prefix conventions like `/app/env/key` for namespace isolation across environments
- Compact periodically — without compaction, revision history grows unboundedly
- Watch channels must be drained — a slow consumer can stall the watch stream
- Leases are cluster-scoped — a lease ID must not be reused after the client reconnects
- Never run etcd with fewer than 3 nodes in production (requires majority for writes)
