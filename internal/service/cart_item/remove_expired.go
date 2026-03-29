package cart_item

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jva44ka/ozon-simulator-go-cart/internal/model"
)

func (s *CartItemService) RemoveExpired(ctx context.Context, reservationId int64) error {
	_, err := s.cartRepository.GetByReservationId(ctx, reservationId)
	if err != nil {
		if errors.Is(err, model.ErrCartItemsNotFound) {
			slog.WarnContext(
				ctx,
				"Trying to remove an already removed reservation",
				"reservationId", reservationId)

			return nil
		}

		return fmt.Errorf("Failed while trying to remove expired reserved cart item: %w", err)
	}

	err = s.cartRepository.RemoveByReservationId(ctx, reservationId)
	if err != nil {
		return fmt.Errorf("Failed while trying to remove expired reserved cart item: %w", err)
	}

	return nil
}
