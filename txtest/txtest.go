package txtest

import (
	"context"

	"github.com/myronrotter/tx"
)

type Manager struct{}

var _ tx.Manager = &Manager{}

func New() *Manager {
	return &Manager{}
}

func (m *Manager) DoInTransaction(ctx context.Context, uow func(ctx context.Context) error) error {
	return uow(ctx)
}
