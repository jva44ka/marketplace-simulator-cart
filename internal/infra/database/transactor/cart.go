package transactor

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jva44ka/marketplace-simulator-cart/internal/infra/database/repository"
	uc "github.com/jva44ka/marketplace-simulator-cart/internal/usecases"
)

type CartServiceTransactor struct {
	pool            *pgxpool.Pool
	cartItemMetrics repository.CartItemRepositoryMetrics
}

func NewCartServiceTransactor(
	pool *pgxpool.Pool,
	cartItemMetrics repository.CartItemRepositoryMetrics,
) *CartServiceTransactor {
	return &CartServiceTransactor{pool: pool, cartItemMetrics: cartItemMetrics}
}

func (t *CartServiceTransactor) InTransaction(
	ctx context.Context,
	fn func(cartItems uc.TxCartItemRepository, outbox uc.TxOutboxRepository) error,
) error {
	return pgx.BeginTxFunc(ctx, t.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return fn(
			repository.NewPgxCartItemTxRepository(tx, t.cartItemMetrics),
			repository.NewReservationConfirmationOutboxTxRepository(tx),
		)
	})
}
