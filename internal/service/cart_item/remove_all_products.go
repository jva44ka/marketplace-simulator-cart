package cart_item

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (s *CartItemService) RemoveAllProducts(ctx context.Context, userId uuid.UUID) error {
	cartItems, err := s.cartRepository.GetByUserId(ctx, userId)
	if err != nil {
		return fmt.Errorf("cartRepository.GetByUserId: %w", err)
	}

	if len(cartItems) > 0 {
		ids := make([]int64, 0, len(cartItems))
		for _, item := range cartItems {
			if item.ReservationId != 0 {
				ids = append(ids, item.ReservationId)
			}
		}
		if len(ids) > 0 {
			if err = s.productClient.ReleaseReservation(ctx, ids); err != nil {
				return fmt.Errorf("productClient.ReleaseReservation: %w", err)
			}
		}
	}

	return s.cartRepository.RemoveByUserId(ctx, userId)
}
