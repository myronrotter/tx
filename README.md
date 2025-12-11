# tx

A Go package for managing nested transactions across GORM and Redis.

## Usage

### GORM Transactions

The `txgorm` package provides transaction management for GORM database operations.

```go
package main

import (
    "context"
    "tx/txgorm"
    "gorm.io/gorm"
)

func main() {
    // Initialize your GORM database connection
    db, _ := gorm.Open(...)
    
    // Create a transaction manager
    manager := txgorm.New(db)
    
    // Execute operations within a transaction
    err := manager.DoInTransaction(context.Background(), func(ctx context.Context) error {
        // Get the transactional DB from context
        db := txgorm.MustGetDB(ctx)
        
        // Perform database operations
        if err := db.Create(&User{Name: "Alice"}).Error; err != nil {
            return err // Automatically rolls back
        }
        
        // Nested transactions share the same transaction context
        err := manager.DoInTransaction(ctx, func(ctx context.Context) error {
            db := txgorm.MustGetDB(ctx)
            return db.Create(&User{Name: "Bob"}).Error
        })
        if err != nil {
            return err // Rolls back all operations
        }
        
        return nil // Commits the transaction
    })
}
```

**Key Points:**
- Use `txgorm.MustGetDB(ctx)` to retrieve the transactional DB instance
- Returning an error triggers automatic rollback
- Returning `nil` commits the transaction
- Nested calls reuse the same transaction

### Redis Transactions

The `txredis` package provides transaction management for Redis using pipelines.

```go
package main

import (
    "context"
    "tx/txredis"
    "github.com/go-redis/redis/v8"
)

func main() {
    // Initialize your Redis client
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    
    // Create a transaction manager
    manager := txredis.New(client)
    
    // Execute operations within a transaction
    err := manager.DoInTransaction(context.Background(), func(ctx context.Context) error {
        // Get the pipeline from context
        pipe := txredis.MustGetPipe(ctx)
        
        // Queue Redis operations
        if err := pipe.Set(ctx, "key1", "value1", 0).Err(); err != nil {
            return err // Automatically discards pipeline
        }
        
        // Nested transactions share the same pipeline
        err := manager.DoInTransaction(ctx, func(ctx context.Context) error {
            pipe := txredis.MustGetPipe(ctx)
            return pipe.Set(ctx, "key2", "value2", 0).Err()
        })
        if err != nil {
            return err // Discards all operations
        }
        
        return nil // Executes the pipeline
    })
}
```

**Key Points:**
- Use `txredis.MustGetPipe(ctx)` to retrieve the Redis pipeline
- Operations are queued, not executed immediately
- Returning an error discards all queued operations
- Returning `nil` executes all operations atomically
- Nested calls reuse the same pipeline
