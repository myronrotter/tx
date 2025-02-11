package txredis_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
	"tx/txredis"

	gredis "github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

type env struct {
	redis  *redis.RedisContainer
	client *gredis.Client
}

func newEnv(ctx context.Context) (*env, error) {
	redisContainer, err := redis.Run(ctx,
		"redis:7",
	)
	if err != nil {
		return nil, err
	}

	connStr, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		return nil, err
	}
	fmt.Printf("connection: %s\n", connStr)
	opts, err := gredis.ParseURL(connStr)
	if err != nil {
		return nil, err
	}
	fmt.Printf("opts: %+v\n", opts)
	client := gredis.NewClient(opts)

	sctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(sctx).Err(); err != nil {
		return nil, err
	}

	return &env{
		redis:  redisContainer,
		client: client,
	}, nil
}

func TestManager(t *testing.T) {
	ctx := context.Background()
	e, err := newEnv(ctx)
	require.NoError(t, err)
	defer func() {
		if err := e.redis.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}()

	manager := txredis.New(e.client)

	t.Run("commit successfully", func(t *testing.T) {
		err := manager.DoInTransaction(context.Background(), func(ctx context.Context) error {
			err := txredis.MustGetPipe(ctx).Set(ctx, "key1", "value1", 0).Err()
			if err != nil {
				return err
			}

			err = manager.DoInTransaction(ctx, func(ctx context.Context) error {
				err = txredis.MustGetPipe(ctx).Set(ctx, "key2", "value2", 0).Err()
				if err != nil {
					return err
				}

				return nil
			})
			if err != nil {
				return err
			}

			return nil
		})
		require.NoError(t, err)

		val1, err := e.client.Get(ctx, "key1").Result()
		assert.NoError(t, err)
		assert.Equal(t, "value1", val1)
		val2, err := e.client.Get(ctx, "key2").Result()
		assert.NoError(t, err)
		assert.Equal(t, "value2", val2)
	})

	t.Run("rollback successfully", func(t *testing.T) {
		err := manager.DoInTransaction(context.Background(), func(ctx context.Context) error {
			err := txredis.MustGetPipe(ctx).Set(ctx, "key3", "value3", 0).Err()
			if err != nil {
				return err
			}

			err = manager.DoInTransaction(ctx, func(ctx context.Context) error {
				return errors.New("test")
			})
			if err != nil {
				return err
			}

			return nil
		})
		require.Error(t, err)

		val3, err := e.client.Get(ctx, "key3").Result()
		assert.Error(t, err)
		assert.ErrorIs(t, err, gredis.Nil)
		assert.Equal(t, "", val3)
	})
}
