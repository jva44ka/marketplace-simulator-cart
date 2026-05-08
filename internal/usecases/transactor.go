package usecases

import (
	"context"

	"github.com/google/uuid"
	"github.com/jva44ka/marketplace-simulator-cart/internal/model"
)

type CartItemRepository interface {
	Create(ctx context.Context, cartItem model.CartItem) (uint64, error)
	Update(ctx context.Context, id uint64, cartItem model.CartItem) error
	GetByUserId(ctx context.Context, userId uuid.UUID) ([]model.CartItem, error)
	GetByUserIdAndSku(ctx context.Context, userId uuid.UUID, sku uint64) (*model.CartItem, error)
	RemoveByUserIdAndSku(ctx context.Context, userId uuid.UUID, sku uint64) error
	RemoveByUserId(ctx context.Context, userId uuid.UUID) error
}

type ProductRepository interface {
	GetProductBySku(ctx context.Context, sku uint64) (model.Product, error)
	AddProduct(ctx context.Context, product model.Product) (*model.Product, error)
}

type ProductClient interface {
	GetBySku(ctx context.Context, sku uint64) (*model.Product, error)
	Reserve(ctx context.Context, productCountsBySkus map[uint64]uint32) (map[uint64]int64, error)
	ReleaseReservation(ctx context.Context, reservationIds []int64) error
}

type TxCartItemRepository interface {
	RemoveByUserId(ctx context.Context, userId uuid.UUID) error
}

type TxOutboxRepository interface {
	Create(ctx context.Context, rec model.ReservationConfirmationOutboxRecordNew) error
}

type Transactor interface {
	InTransaction(ctx context.Context, fn func(
		cartItems TxCartItemRepository,
		outbox TxOutboxRepository,
	) error) error
}

type RecordBuilder interface {
	BuildRecords(ctx context.Context, cartItems []model.CartItem, reservationIds map[uint64]int64) ([]model.ReservationConfirmationOutboxRecordNew, error)
}

type CheckoutMetrics interface {
	RecordSuccess(totalPrice float64)
	RecordFailure(reason string)
}
