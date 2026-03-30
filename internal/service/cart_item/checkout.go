package cart_item

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/model"
)

func (s *CartItemService) Checkout(ctx context.Context, userId uuid.UUID) (float64, error) {
	cartItems, err := s.cartItemRepository.GetByUserId(ctx, userId)
	if err != nil {
		return 0.0, fmt.Errorf("cartRepository.GetByUserId: %w", err)
	}

	if len(cartItems) == 0 {
		return 0.0, model.ErrCartEmpty
	}

	skuCounts := make(map[uint64]uint32, len(cartItems))
	totalPrice := 0.0
	for _, item := range cartItems {
		skuCounts[item.Product.Sku] = item.Count
		totalPrice += item.Product.Price * float64(item.Count)
	}

	reservationIds, err := s.productClient.Reserve(ctx, skuCounts)
	if err != nil {
		return 0.0, fmt.Errorf("productClient.Reserve: %w", err)
	}

	if err = s.cartItemRepository.RemoveByUserId(ctx, userId); err != nil {
		ids := reservationIdsToSlice(reservationIds)
		if releaseErr := s.productClient.ReleaseReservation(ctx, ids); releaseErr != nil {
			slog.ErrorContext(ctx, "failed to release reservations after cart remove error", "err", releaseErr)
		}
		return 0.0, fmt.Errorf("cartRepository.RemoveByUserId: %w", err)
	}

	// TODO: заменить на outbox
	go func() {
		ids := reservationIdsToSlice(reservationIds)
		if err := s.productClient.ConfirmReservation(context.Background(), ids); err != nil {
			slog.Error("failed to confirm reservations", "err", err)
		}
	}()

	return totalPrice, nil
}

func reservationIdsToSlice(m map[uint64]int64) []int64 {
	ids := make([]int64, 0, len(m))
	for _, id := range m {
		ids = append(ids, id)
	}
	return ids
}
