package cart_item

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/model"
)

func (s *CartItemService) Checkout(ctx context.Context, userId uuid.UUID) (float64, error) {
	cartItems, err := s.cartRepository.GetByUserId(ctx, userId)
	if err != nil {
		return 0.0, fmt.Errorf("cartRepository.GetByUserId: %w", err)
	}

	if len(cartItems) == 0 {
		return 0.0, model.ErrCartEmpty
	}

	ids := make([]int64, 0, len(cartItems))
	totalPrice := 0.0
	for _, item := range cartItems {
		if item.ReservationId != 0 {
			ids = append(ids, item.ReservationId)
		}
		totalPrice += item.Product.Price
	}

	if err = s.productClient.ConfirmReservation(ctx, ids); err != nil {
		return 0.0, fmt.Errorf("productClient.ConfirmReservation: %w", err)
	}

	if err = s.cartRepository.RemoveByUserId(ctx, userId); err != nil {
		return 0.0, fmt.Errorf("cartRepository.RemoveByUserId: %w", err)
	}

	return totalPrice, nil
}
