package rcpostgres

import (
	"context"

	"gorm.io/gorm"
)

type contextKeyTx struct{}

// getTxFromCtx извлекает *gorm.DB из контекста.
// Если транзакции нет, возвращает nil.
func getTxFromCtx(ctx context.Context) *gorm.DB {
	if v := ctx.Value(contextKeyTx{}); v != nil {
		if tx, ok := v.(*gorm.DB); ok {
			return tx
		}
	}
	return nil
}

// ctxWithTx сохраняет *gorm.DB в контекст.
func ctxWithTx(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, contextKeyTx{}, tx)
}
