package cart_item

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/model"
)

type CartItemTxRepository interface {
	RemoveByUserId(ctx context.Context, userId uuid.UUID) error
}

type OutboxTxRepository interface {
	Create(ctx context.Context, rec model.ReservationConfirmationOutboxRecordNew) error
}

type CartItemRepository interface {
	Create(_ context.Context, cartItem model.CartItem) (uint64, error)
	Update(_ context.Context, id uint64, cartItem model.CartItem) error
	GetByUserId(_ context.Context, userId uuid.UUID) ([]model.CartItem, error)
	GetByUserIdAndSku(_ context.Context, userId uuid.UUID, sku uint64) (*model.CartItem, error)
	RemoveByUserIdAndSku(_ context.Context, userId uuid.UUID, sku uint64) error
	RemoveByUserId(_ context.Context, userId uuid.UUID) error
	WithTx(tx pgx.Tx) CartItemTxRepository
}

type ProductRepository interface {
	GetProductBySku(ctx context.Context, sku uint64) (model.Product, error)
	AddProduct(ctx context.Context, product model.Product) (*model.Product, error)
}

type OutboxRepository interface {
	WithTx(tx pgx.Tx) OutboxTxRepository
}

type DBManager interface {
	CartItemRepo() CartItemRepository
	ProductRepo() ProductRepository
	OutboxRepo() OutboxRepository
	InTransaction(ctx context.Context, fn func(pgx.Tx) error) error
}

type RecordBuilder interface {
	BuildRecords(cartItems []model.CartItem, reservationIds map[uint64]int64) ([]model.ReservationConfirmationOutboxRecordNew, error)
}

type ProductClient interface {
	GetBySku(ctx context.Context, sku uint64) (*model.Product, error)
	Reserve(ctx context.Context, productCountsBySkus map[uint64]uint32) (map[uint64]int64, error)
	ReleaseReservation(ctx context.Context, reservationIds []int64) error
	ConfirmReservation(ctx context.Context, reservationIds []int64) error
}

type CartItemService struct {
	db            DBManager
	productClient ProductClient
	recordBuilder RecordBuilder
}

func NewCartItemService(
	db DBManager,
	productClient ProductClient,
	recordBuilder RecordBuilder,
) *CartItemService {
	return &CartItemService{
		db:            db,
		productClient: productClient,
		recordBuilder: recordBuilder,
	}
}
