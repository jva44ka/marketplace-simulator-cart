package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	outboxContracts "github.com/jva44ka/ozon-simulator-go-cart/api_internal/outbox"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/model"
)

type ReservationConfirmationRecordBuilder struct{}

func NewReservationConfirmationRecordBuilder() *ReservationConfirmationRecordBuilder {
	return &ReservationConfirmationRecordBuilder{}
}

func (b *ReservationConfirmationRecordBuilder) BuildRecords(
	ctx context.Context,
	cartItems []model.CartItem,
	reservationIds map[uint64]int64,
) ([]model.ReservationConfirmationOutboxRecordNew, error) {
	records := make([]model.ReservationConfirmationOutboxRecordNew, 0, len(cartItems))

	for _, item := range cartItems {
		reservationId, ok := reservationIds[item.Product.Sku]
		if !ok {
			slog.ErrorContext(ctx, "reservation id not found for sku", "sku", item.Product.Sku)
			continue
		}

		data := outboxContracts.ReservationConfirmationData{
			ReservationId: reservationId,
		}

		dataBytes, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("ReservationConfirmationRecordBuilder.BuildRecords marshal data: %w", err)
		}

		headers := map[string]string{}
		headersBytes, err := json.Marshal(headers)
		if err != nil {
			return nil, fmt.Errorf("ReservationConfirmationRecordBuilder.BuildRecords marshal headers: %w", err)
		}

		records = append(records, model.ReservationConfirmationOutboxRecordNew{
			Key:     strconv.FormatInt(reservationId, 10),
			Data:    dataBytes,
			Headers: headersBytes,
		})
	}

	return records, nil
}
