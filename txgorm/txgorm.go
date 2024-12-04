package txgorm

import (
	"context"
	"tx"

	"gorm.io/gorm"
)

const contextKey tx.ContextKey = "gorm"

func MustGetDB(ctx context.Context) *gorm.DB {
	db, ok := ctx.Value(contextKey).(*gorm.DB)
	if !ok {
		panic("no *gorm.DB found")
	}

	return db
}

type Manager struct {
	db *gorm.DB
}

var _ tx.Manager = &Manager{}

func New(db *gorm.DB) *Manager {
	return &Manager{
		db: db,
	}
}

func (m *Manager) DoInTransaction(ctx context.Context, uow func(ctx context.Context) error) error {
	db, ok := ctx.Value(contextKey).(*gorm.DB)
	if !ok {
		db = m.db.Begin()
		ctx = context.WithValue(ctx, contextKey, db)
	}

	err := uow(ctx)
	if err != nil {
		db.Rollback()
		return err
	}

	db.Commit()
	return nil
}
