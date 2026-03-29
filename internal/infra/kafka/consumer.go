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

type Consumer struct {
	reader *segkafka.Reader
}

func NewConsumer(brokers []string, topic, groupId string) *Consumer {
	return &Consumer{
		reader: segkafka.NewReader(segkafka.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupId,
		}),
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

		//TODO: do something
		slog.InfoContext(ctx, "recived reservation expired event", "event", event)
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
