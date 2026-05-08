package database

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	cart_item "github.com/jva44ka/marketplace-simulator-cart/internal/service/cart_item"
)

// CartServiceTransactor satisfies service/cart_item.Transactor.
// It opens a pgx transaction, builds the tx-bound repo views, and passes them
// into the lambda so the service never touches pgx.Tx directly.
type CartServiceTransactor struct {
	pool      *pgxpool.Pool
	cartItems *PgxCartItemRepository
	outbox    *ReservationConfirmationOutboxRepository
}

func NewCartServiceTransactor(
	pool *pgxpool.Pool,
	cartItems *PgxCartItemRepository,
	outbox *ReservationConfirmationOutboxRepository,
) *CartServiceTransactor {
	return &CartServiceTransactor{pool: pool, cartItems: cartItems, outbox: outbox}
}

func (t *CartServiceTransactor) InTransaction(
	ctx context.Context,
	fn func(cartItems cart_item.CartItemTxRepo, outbox cart_item.OutboxTxRepo) error,
) error {
	return pgx.BeginTxFunc(ctx, t.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return fn(t.cartItems.WithTx(tx), t.outbox.WithTx(tx))
	})
}
