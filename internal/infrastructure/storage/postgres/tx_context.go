package postgres

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct{}

func getTx(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx
	}
	return nil
}
