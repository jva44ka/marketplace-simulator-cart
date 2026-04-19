package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	outboxContracts "github.com/jva44ka/marketplace-simulator-cart/api_internal/outbox"
	"github.com/jva44ka/marketplace-simulator-cart/internal/model"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
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

	headers := map[string]string{}
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(headers))

	headersBytes, err := json.Marshal(headers)
	if err != nil {
		return nil, fmt.Errorf("ReservationConfirmationRecordBuilder.BuildRecords marshal headers: %w", err)
	}

	for _, item := range cartItems {
		reservationId, ok := reservationIds[item.Product.Sku]
		if !ok {
			slog.ErrorContext(ctx, "reservation id not found for sku", "sku", item.Product.Sku)
			continue
		}

		data := outboxContracts.ReservationConfirmationData{
			ReservationId: reservationId,
		}

		var dataBytes []byte
		dataBytes, err = json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("ReservationConfirmationRecordBuilder.BuildRecords marshal data: %w", err)
		}

		records = append(records, model.ReservationConfirmationOutboxRecordNew{
			Key:     strconv.FormatInt(reservationId, 10),
			Data:    dataBytes,
			Headers: headersBytes,
		})
	}

	return records, nil
}
