package txredis

import (
	"context"
	"errors"
	"tx"

	"github.com/go-redis/redis/v8"
)

const contextKey tx.ContextKey = "redis"

func MustGetPipe(ctx context.Context) redis.Pipeliner {
	pipe, ok := ctx.Value(contextKey).(redis.Pipeliner)
	if !ok {
		panic("no redis.Pipeliner found")
	}

	return pipe
}

type Manager struct {
	client *redis.Client
}

var _ tx.Manager = &Manager{}

func New(cache *redis.Client) *Manager {
	return &Manager{
		client: cache,
	}
}

func (m *Manager) DoInTransaction(ctx context.Context, uow func(ctx context.Context) error) error {
	commiter := false
	pipe, ok := ctx.Value(contextKey).(redis.Pipeliner)
	if !ok {
		pipe = m.client.TxPipeline()
		ctx = context.WithValue(ctx, contextKey, pipe)
		commiter = true
	}

	err := uow(ctx)
	if err != nil {
		discardErr := pipe.Discard()
		if discardErr != nil {
			return errors.Join(err, discardErr)
		}

		return err
	}

	if commiter {
		_, execErr := pipe.Exec(ctx)
		if execErr != nil {
			return execErr
		}
	}
	return nil
}
