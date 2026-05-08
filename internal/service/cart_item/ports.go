package cart_item

import (
	"context"

	"github.com/google/uuid"
	"github.com/jva44ka/marketplace-simulator-cart/internal/model"
)

// Transactor opens a transaction and calls fn with pre-bound tx repos.
// The service never sees pgx.Tx.
type Transactor interface {
	InTransaction(ctx context.Context, fn func(
		cartItems CartItemTxRepo,
		outbox    OutboxTxRepo,
	) error) error
}

// CartItemTxRepo contains cart-item writes that run inside a transaction.
type CartItemTxRepo interface {
	RemoveByUserId(ctx context.Context, userId uuid.UUID) error
}

// CartItemRepository is the full read/write interface for cart item persistence.
type CartItemRepository interface {
	Create(ctx context.Context, cartItem model.CartItem) (uint64, error)
	Update(ctx context.Context, id uint64, cartItem model.CartItem) error
	GetByUserId(ctx context.Context, userId uuid.UUID) ([]model.CartItem, error)
	GetByUserIdAndSku(ctx context.Context, userId uuid.UUID, sku uint64) (*model.CartItem, error)
	RemoveByUserIdAndSku(ctx context.Context, userId uuid.UUID, sku uint64) error
	RemoveByUserId(ctx context.Context, userId uuid.UUID) error
	CountActiveCarts(ctx context.Context) (int64, error)
	CountCartItems(ctx context.Context) (int64, error)
}

// LocalProductRepository is the cart-local product cache.
type LocalProductRepository interface {
	GetProductBySku(ctx context.Context, sku uint64) (model.Product, error)
	AddProduct(ctx context.Context, product model.Product) (*model.Product, error)
}

// OutboxTxRepo contains the outbox write that must run inside a transaction.
type OutboxTxRepo interface {
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
}
