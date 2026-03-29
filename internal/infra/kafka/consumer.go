package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	segkafka "github.com/segmentio/kafka-go"
)

type ReservationExpiredEvent struct {
	ReservationId int64  `json:"reservation_id"`
	Sku           uint64 `json:"sku"`
	Count         uint32 `json:"count"`
}

type CartItemService interface {
	RemoveExpired(ctx context.Context, reservationId int64) error
}

type Consumer struct {
	reader          *segkafka.Reader
	cartItemService CartItemService
}

func NewConsumer(brokers []string, topic, groupId string, cartItemService CartItemService) *Consumer {
	return &Consumer{
		reader: segkafka.NewReader(segkafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupId,
		}),
		cartItemService: cartItemService,
	}
}

func (c *Consumer) Run(ctx context.Context) {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.ErrorContext(ctx, "kafka consumer read error", "err", err)
			continue
		}

		var event ReservationExpiredEvent
		if err = json.Unmarshal(msg.Value, &event); err != nil {
			slog.ErrorContext(ctx, "failed to unmarshal reservation expired event", "err", err)
			continue
		}

		if err = c.cartItemService.RemoveExpired(ctx, event.ReservationId); err != nil {
			slog.ErrorContext(ctx, "failed to remove expired cart item",
				"reservation_id", event.ReservationId, "err", err)
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
