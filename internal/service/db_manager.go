package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jva44ka/marketplace-simulator-cart/internal/model"
)

type CartItemRepository interface {
	Create(ctx context.Context, cartItem model.CartItem) (uint64, error)
	Update(ctx context.Context, id uint64, cartItem model.CartItem) error
	GetByUserId(ctx context.Context, userId uuid.UUID) ([]model.CartItem, error)
	GetByUserIdAndSku(ctx context.Context, userId uuid.UUID, sku uint64) (*model.CartItem, error)
	RemoveByUserIdAndSku(ctx context.Context, userId uuid.UUID, sku uint64) error
	RemoveByUserId(ctx context.Context, userId uuid.UUID) error
	WithTx(tx pgx.Tx) CartItemTxRepository
}

type CartItemTxRepository interface {
	RemoveByUserId(ctx context.Context, userId uuid.UUID) error
}

type ProductRepository interface {
	GetProductBySku(ctx context.Context, sku uint64) (model.Product, error)
	AddProduct(ctx context.Context, product model.Product) (*model.Product, error)
}

type OutboxRepository interface {
	WithTx(tx pgx.Tx) OutboxTxRepository
}

type OutboxTxRepository interface {
	Create(ctx context.Context, rec model.ReservationConfirmationOutboxRecordNew) error
}

type DBManager interface {
	CartItemRepo() CartItemRepository
	ProductRepo() ProductRepository
	OutboxRepo() OutboxRepository
	InTransaction(ctx context.Context, fn func(pgx.Tx) error) error
}
