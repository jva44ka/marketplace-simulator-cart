package cart_item

import (
	"context"

	"github.com/jva44ka/ozon-simulator-go-cart/internal/model"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/service"
)

type RecordBuilder interface {
	BuildRecords(cartItems []model.CartItem, reservationIds map[uint64]int64) ([]model.ReservationConfirmationOutboxRecordNew, error)
}

type ProductClient interface {
	GetBySku(ctx context.Context, sku uint64) (*model.Product, error)
	Reserve(ctx context.Context, productCountsBySkus map[uint64]uint32) (map[uint64]int64, error)
	ReleaseReservation(ctx context.Context, reservationIds []int64) error
}

type CartItemService struct {
	db            service.DBManager
	productClient ProductClient
	recordBuilder RecordBuilder
}

func NewCartItemService(
	db service.DBManager,
	productClient ProductClient,
	recordBuilder RecordBuilder,
) *CartItemService {
	return &CartItemService{
		db:            db,
		productClient: productClient,
		recordBuilder: recordBuilder,
	}
}
