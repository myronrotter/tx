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
	*gorm.DB
}

var _ tx.Manager = &Manager{}

func New(db *gorm.DB) *Manager {
	return &Manager{
		DB: db,
	}
}

func (m *Manager) DoInTransaction(ctx context.Context, uow func(ctx context.Context) error) error {
	c := ctx

	db, ok := ctx.Value(contextKey).(*gorm.DB)
	if !ok {
		db = m.Begin()
		c = context.WithValue(ctx, contextKey, db)
	}

	err := uow(c)
	if err != nil {
		db.Rollback()
		return err
	}

	db.Commit()
	return nil
}
