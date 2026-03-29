package cart_item

import (
	"context"

	"github.com/google/uuid"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/model"
)

type CartItemRepository interface {
	Create(_ context.Context, cartItem model.CartItem) (uint64, error)
	Update(_ context.Context, id uint64, cartItem model.CartItem) error
	GetByUserId(_ context.Context, userId uuid.UUID) ([]model.CartItem, error)
	GetByUserIdAndSku(_ context.Context, userId uuid.UUID, sku uint64) (*model.CartItem, error)
	RemoveByUserIdAndSku(_ context.Context, userId uuid.UUID, sku uint64) error
	RemoveByUserId(_ context.Context, userId uuid.UUID) error
}

type ProductRepository interface {
	GetProductBySku(ctx context.Context, sku uint64) (model.Product, error)
	AddProduct(ctx context.Context, product model.Product) (*model.Product, error)
}

type ProductClient interface {
	GetBySku(ctx context.Context, sku uint64) (*model.Product, error)
	Reserve(ctx context.Context, productCountsBySkus map[uint64]uint32) (map[uint64]int64, error)
	ReleaseReservation(ctx context.Context, reservationIds []int64) error
	ConfirmReservation(ctx context.Context, reservationIds []int64) error
}

type CartItemService struct {
	cartItemRepository CartItemRepository
	productClient      ProductClient
	productRepository  ProductRepository
}

func NewCartItemService(
	cartItemRepository CartItemRepository,
	productClient ProductClient,
	productRepository ProductRepository,
) *CartItemService {
	return &CartItemService{
		cartItemRepository: cartItemRepository,
		productClient:      productClient,
		productRepository:  productRepository,
	}
}
