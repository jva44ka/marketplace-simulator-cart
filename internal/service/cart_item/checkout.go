package cart_item

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/model"
)

func (s *CartItemService) Checkout(ctx context.Context, userId uuid.UUID) (float64, error) {
	cartItems, err := s.db.CartItemRepo().GetByUserId(ctx, userId)
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

	outboxRecords, err := s.recordBuilder.BuildRecords(ctx, cartItems, reservationIds)
	if err != nil {
		releaseErr := s.productClient.ReleaseReservation(ctx, reservationIdsToSlice(reservationIds))
		if releaseErr != nil {
			return 0.0, fmt.Errorf("checkout transaction failed: %w; release also failed: %v", err, releaseErr)
		}

		return 0.0, fmt.Errorf("recordBuilder.BuildRecords: %w", err)
	}

	err = s.db.InTransaction(ctx, func(tx pgx.Tx) error {
		if err = s.db.CartItemRepo().WithTx(tx).RemoveByUserId(ctx, userId); err != nil {
			return fmt.Errorf("cartItemTxRepo.RemoveByUserId: %w", err)
		}
		outboxTxRepo := s.db.OutboxRepo().WithTx(tx)
		for _, rec := range outboxRecords {
			if err = outboxTxRepo.Create(ctx, rec); err != nil {
				return fmt.Errorf("outboxTxRepo.Create: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		releaseErr := s.productClient.ReleaseReservation(ctx, reservationIdsToSlice(reservationIds))
		if releaseErr != nil {
			return 0.0, fmt.Errorf("checkout transaction failed: %w; release also failed: %v", err, releaseErr)
		}

		return 0.0, fmt.Errorf("checkout transaction: %w", err)
	}

	return totalPrice, nil
}

func reservationIdsToSlice(m map[uint64]int64) []int64 {
	ids := make([]int64, 0, len(m))
	for _, id := range m {
		ids = append(ids, id)
	}
	return ids
}
