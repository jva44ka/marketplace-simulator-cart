package cart_item

import (
	"context"

	"github.com/google/uuid"
	"github.com/jva44ka/marketplace-simulator-cart/internal/model"
)

type TxCartItemRepository interface {
	RemoveByUserId(ctx context.Context, userId uuid.UUID) error
}

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

type ProductRepository interface {
	GetProductBySku(ctx context.Context, sku uint64) (model.Product, error)
	AddProduct(ctx context.Context, product model.Product) (*model.Product, error)
}

type TxOutboxRepository interface {
	Create(ctx context.Context, rec model.ReservationConfirmationOutboxRecordNew) error
}

type OutboxRepository interface {
	GetPending(ctx context.Context, limit int) ([]model.ReservationConfirmationOutboxRecord, error)
	CountPending(ctx context.Context) (int64, error)
	CountDeadLetters(ctx context.Context) (int64, error)
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
	IncrementRetry(ctx context.Context, id uuid.UUID) error
	MarkDeadLetter(ctx context.Context, id uuid.UUID, reason string) error
}

type Transactor interface {
	InTransaction(ctx context.Context, fn func(
		cartItemRepo TxCartItemRepository,
		outboxRepo TxOutboxRepository,
	) error) error
}
