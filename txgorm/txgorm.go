package txgorm

import (
	"context"
	"errors"
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
	commiter := false
	db, ok := ctx.Value(contextKey).(*gorm.DB)
	if !ok {
		db = m.db.Begin()
		ctx = context.WithValue(ctx, contextKey, db)
		commiter = true
	}

	err := uow(ctx)
	if err != nil {
		rollbackErr := db.Rollback().Error
		if rollbackErr != nil {
			return errors.Join(err, rollbackErr)
		}

		return err
	}

	if commiter {
		commitErr := db.Commit().Error
		if commitErr != nil {
			return commitErr
		}
	}
	return nil
}
