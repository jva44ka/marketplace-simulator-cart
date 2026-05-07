package cart_item

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jva44ka/marketplace-simulator-cart/internal/model"
)

// Transactor begins a DB transaction and hands the live pgx.Tx to the caller.
// Inside fn, call repo.WithTx(tx) to bind repositories to the transaction.
type Transactor interface {
	InTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error
}

// CartItemTxRepository contains the cart-item writes that must run inside a transaction.
type CartItemTxRepository interface {
	RemoveByUserId(ctx context.Context, userId uuid.UUID) error
}

// CartItemRepository is the full interface for cart item persistence.
type CartItemRepository interface {
	Create(ctx context.Context, cartItem model.CartItem) (uint64, error)
	Update(ctx context.Context, id uint64, cartItem model.CartItem) error
	GetByUserId(ctx context.Context, userId uuid.UUID) ([]model.CartItem, error)
	GetByUserIdAndSku(ctx context.Context, userId uuid.UUID, sku uint64) (*model.CartItem, error)
	RemoveByUserIdAndSku(ctx context.Context, userId uuid.UUID, sku uint64) error
	RemoveByUserId(ctx context.Context, userId uuid.UUID) error
	CountActiveCarts(ctx context.Context) (int64, error)
	CountCartItems(ctx context.Context) (int64, error)
	WithTx(tx pgx.Tx) CartItemTxRepository
}

// LocalProductRepository is the cart-local product cache.
type LocalProductRepository interface {
	GetProductBySku(ctx context.Context, sku uint64) (model.Product, error)
	AddProduct(ctx context.Context, product model.Product) (*model.Product, error)
}

// OutboxTxRepository contains the outbox write that must run inside a transaction.
type OutboxTxRepository interface {
	Create(ctx context.Context, rec model.ReservationConfirmationOutboxRecordNew) error
}

// OutboxRepository is the full interface for reservation outbox persistence.
type OutboxRepository interface {
	GetPending(ctx context.Context, limit int) ([]model.ReservationConfirmationOutboxRecord, error)
	CountPending(ctx context.Context) (int64, error)
	CountDeadLetters(ctx context.Context) (int64, error)
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
	IncrementRetry(ctx context.Context, id uuid.UUID) error
	MarkDeadLetter(ctx context.Context, id uuid.UUID, reason string) error
	WithTx(tx pgx.Tx) OutboxTxRepository
}
