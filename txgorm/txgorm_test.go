package txgorm_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"tx/txgorm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	gmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type env struct {
	mysql *mysql.MySQLContainer
	db    *gorm.DB
}

func newEnv(ctx context.Context) (*env, error) {
	mysqlContainer, err := mysql.Run(ctx,
		"mysql:8.0",
		mysql.WithDatabase("db"),
		mysql.WithUsername("user"),
		mysql.WithPassword("password"),
	)
	if err != nil {
		return nil, err
	}

	p, err := mysqlContainer.MappedPort(ctx, "3306")
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("user:password@tcp(localhost:%s)/db?charset=utf8mb4&parseTime=True&loc=Local", p.Port())
	fmt.Printf("DSN: %s", dsn)
	db, err := gorm.Open(gmysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return &env{
		mysql: mysqlContainer,
		db:    db,
	}, nil
}

func TestManager(t *testing.T) {
	type User struct {
		Id   uint64 `gorm:"primaryKey,autoIncrement"`
		Name string
	}

	ctx := context.Background()
	e, err := newEnv(ctx)
	require.NoError(t, err)
	defer func() {
		if err := e.mysql.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %s", err)
		}
	}()

	err = e.db.AutoMigrate(&User{})
	require.NoError(t, err)

	manager := txgorm.New(e.db)

	t.Run("commit successfully", func(t *testing.T) {
		users1 := []User{
			{
				Name: "Mark1",
			},
			{
				Name: "Peter1",
			},
		}

		users2 := []User{
			{
				Name: "Mark2",
			},
			{
				Name: "Peter2",
			},
		}

		err := manager.DoInTransaction(context.Background(), func(ctx context.Context) error {
			err := txgorm.MustGetDB(ctx).Create(users1).Error
			if err != nil {
				return err
			}

			err = manager.DoInTransaction(context.Background(), func(ctx context.Context) error {
				err = txgorm.MustGetDB(ctx).Create(users2).Error
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

		var actuals []User
		err = e.db.Model(&User{}).Find(&actuals).Error
		assert.NoError(t, err)
		assert.Len(t, actuals, 4)
	})

	t.Run("rollback successfully", func(t *testing.T) {
		users := []User{
			{
				Name: "Griffin",
			},
		}

		err := manager.DoInTransaction(context.Background(), func(ctx context.Context) error {
			err := txgorm.MustGetDB(ctx).Create(users).Error
			if err != nil {
				return err
			}

			err = manager.DoInTransaction(context.Background(), func(ctx context.Context) error {
				return errors.New("test")
			})
			if err != nil {
				return err
			}

			return nil
		})
		require.Error(t, err)

		var actuals []User
		err = e.db.Model(&User{}).Where("name = ?", users[0].Name).Find(&actuals).Error
		assert.NoError(t, err)
		assert.Len(t, actuals, 0)
	})
}
