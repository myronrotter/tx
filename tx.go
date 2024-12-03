package tx

import "context"

type ContextKey string

type Manager interface {
	DoInTransaction(context.Context, func(ctx context.Context) error) error
}
