package cart_item

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (s *CartItemService) RemoveProduct(ctx context.Context, userId uuid.UUID, sku uint64) error {
	cartItem, err := s.cartRepository.GetByUserIdAndSku(ctx, userId, sku)
	if err != nil {
		return fmt.Errorf("cartRepository.GetByUserIdAndSku: %w", err)
	}

	if cartItem.ReservationId != 0 {
		if err = s.productClient.ReleaseReservation(ctx, []int64{cartItem.ReservationId}); err != nil {
			return fmt.Errorf("productClient.ReleaseReservation: %w", err)
		}
	}

	return s.cartRepository.RemoveByUserIdAndSku(ctx, userId, sku)
}
