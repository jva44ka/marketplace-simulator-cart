package usecases

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jva44ka/marketplace-simulator-cart/internal/model"
)

type CheckoutUseCase struct {
	transactor    Transactor
	cartItems     CartItemRepository
	productClient ProductClient
	recordBuilder RecordBuilder
	metrics       CheckoutMetrics
}

func NewCheckoutUseCase(
	transactor Transactor,
	cartItems CartItemRepository,
	productClient ProductClient,
	recordBuilder RecordBuilder,
	metrics CheckoutMetrics,
) *CheckoutUseCase {
	return &CheckoutUseCase{
		transactor:    transactor,
		cartItems:     cartItems,
		productClient: productClient,
		recordBuilder: recordBuilder,
		metrics:       metrics,
	}
}

func (uc *CheckoutUseCase) Execute(ctx context.Context, userId uuid.UUID) (float64, error) {
	cartItems, err := uc.cartItems.GetByUserId(ctx, userId)
	if err != nil {
		return 0.0, fmt.Errorf("cartItemRepository.GetByUserId: %w", err)
	}

	if len(cartItems) == 0 {
		uc.metrics.RecordFailure("empty_cart")
		return 0.0, model.ErrCartEmpty
	}

	skuCounts := make(map[uint64]uint32, len(cartItems))
	totalPrice := 0.0
	for _, item := range cartItems {
		skuCounts[item.Product.Sku] = item.Count
		totalPrice += item.Product.Price * float64(item.Count)
	}

	reservationIds, err := uc.productClient.Reserve(ctx, skuCounts)
	if err != nil {
		uc.metrics.RecordFailure(checkoutFailureReason(err))
		return 0.0, fmt.Errorf("productClient.Reserve: %w", err)
	}

	outboxRecords, err := uc.recordBuilder.BuildRecords(ctx, cartItems, reservationIds)
	if err != nil {
		releaseErr := uc.productClient.ReleaseReservation(ctx, reservationIdsToSlice(reservationIds))
		if releaseErr != nil {
			uc.metrics.RecordFailure("internal")
			return 0.0, fmt.Errorf("checkout transaction failed: %w; release also failed: %v", err, releaseErr)
		}
		uc.metrics.RecordFailure("internal")
		return 0.0, fmt.Errorf("recordBuilder.BuildRecords: %w", err)
	}

	err = uc.transactor.InTransaction(ctx, func(txCartItems TxCartItemRepository, txOutbox TxOutboxRepository) error {
		if err = txCartItems.RemoveByUserId(ctx, userId); err != nil {
			return fmt.Errorf("cartItemRepository.RemoveByUserId: %w", err)
		}
		for _, rec := range outboxRecords {
			if err = txOutbox.Create(ctx, rec); err != nil {
				return fmt.Errorf("outbox.Create: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		releaseErr := uc.productClient.ReleaseReservation(ctx, reservationIdsToSlice(reservationIds))
		if releaseErr != nil {
			uc.metrics.RecordFailure("internal")
			return 0.0, fmt.Errorf("checkout transaction failed: %w; release also failed: %v", err, releaseErr)
		}
		uc.metrics.RecordFailure("internal")
		return 0.0, fmt.Errorf("checkout transaction: %w", err)
	}

	uc.metrics.RecordSuccess(totalPrice)
	return totalPrice, nil
}

func checkoutFailureReason(err error) string {
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "insufficient"):
		return "insufficient_stock"
	case strings.Contains(msg, "not found"):
		return "product_not_found"
	default:
		return "internal"
	}
}

func reservationIdsToSlice(m map[uint64]int64) []int64 {
	ids := make([]int64, 0, len(m))
	for _, id := range m {
		ids = append(ids, id)
	}
	return ids
}
