package jobs

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	outboxContracts "github.com/jva44ka/marketplace-simulator-cart/api_internal/outbox"
	"github.com/jva44ka/marketplace-simulator-cart/internal/infra/config"
	"github.com/jva44ka/marketplace-simulator-cart/internal/model"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type OutboxRepository interface {
	GetPending(ctx context.Context, limit int) ([]model.ReservationConfirmationOutboxRecord, error)
	DeleteBatch(ctx context.Context, ids []uuid.UUID) error
	IncrementRetry(ctx context.Context, id uuid.UUID) error
	MarkDeadLetter(ctx context.Context, id uuid.UUID, reason string) error
}

type ProductsClient interface {
	ConfirmReservation(ctx context.Context, reservationIds []int64) error
}

type OutboxJobMetrics interface {
	ReportProcessed(status string, count int)
	ReportTickDuration(d time.Duration)
	ReportRecordAge(age time.Duration)
}

type processBatchResult struct {
	SuccessRecords      []uuid.UUID
	FailedRecordReasons map[uuid.UUID]string
}

type ReservationConfirmationOutboxJob struct {
	outboxRepo     OutboxRepository
	productsClient ProductsClient
	metrics        OutboxJobMetrics
	cfgStore       *config.ConfigStore
	tracer         trace.Tracer
}

func NewReservationConfirmationOutboxJob(
	outboxRepo OutboxRepository,
	productsClient ProductsClient,
	metrics OutboxJobMetrics,
	cfgStore *config.ConfigStore,
) *ReservationConfirmationOutboxJob {
	return &ReservationConfirmationOutboxJob{
		outboxRepo:     outboxRepo,
		productsClient: productsClient,
		metrics:        metrics,
		cfgStore:       cfgStore,
		tracer:         otel.Tracer("cart-outbox"),
	}
}

func (j *ReservationConfirmationOutboxJob) Run(ctx context.Context) {
	lastProcessed := 0

	for {
		cfg := j.cfgStore.Load().Jobs.ReservationConfirmationOutbox

		//todo рефакторинг, чтобы не парсить на каждую итерацию
		idleInterval, err := time.ParseDuration(cfg.IdleInterval)
		if err != nil {
			idleInterval = 100 * time.Millisecond
			slog.Warn("ReservationConfirmationOutboxJob: invalid idle-interval, using 100ms", "err", err)
		}

		activeInterval, err := time.ParseDuration(cfg.ActiveInterval)
		if err != nil {
			activeInterval = 0
			slog.Warn("ReservationConfirmationOutboxJob: invalid active-interval, using 0", "err", err)
		}

		interval := idleInterval
		if lastProcessed > 0 {
			interval = activeInterval
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}

		if !cfg.Enabled {
			lastProcessed = 0
			continue
		}

		lastProcessed = j.tick(ctx, cfg.BatchSize, cfg.MaxRetries)
	}
}

func (j *ReservationConfirmationOutboxJob) tick(ctx context.Context, batchSize int, maxRetryCount int) int {
	tickStart := time.Now()
	defer func() {
		j.metrics.ReportTickDuration(time.Since(tickStart))
	}()

	records, err := j.outboxRepo.GetPending(ctx, batchSize)
	if err != nil {
		slog.ErrorContext(ctx, "ReservationConfirmationOutboxJob: GetPending failed", "err", err)
		return 0
	}

	if len(records) == 0 {
		return 0
	}

	// Замеряем возраст записей
	for _, record := range records {
		j.metrics.ReportRecordAge(time.Since(record.CreatedAt))
	}

	batchResult := j.processBatch(ctx, records)

	deadLetterCount := 0
	failedCount := 0

	for id, reason := range batchResult.FailedRecordReasons {
		rec := findRecord(records, id)
		if rec == nil {
			continue
		}

		if rec.RetryCount+1 >= maxRetryCount {
			if dlErr := j.outboxRepo.MarkDeadLetter(ctx, id, reason); dlErr != nil {
				slog.ErrorContext(ctx, "ReservationConfirmationOutboxJob: MarkDeadLetter failed", "id", id, "err", dlErr)
			}
			deadLetterCount++
		} else {
			if retryErr := j.outboxRepo.IncrementRetry(ctx, id); retryErr != nil {
				slog.ErrorContext(ctx, "ReservationConfirmationOutboxJob: IncrementRetry failed", "id", id, "err", retryErr)
			}
			failedCount++
		}
	}

	if len(batchResult.SuccessRecords) > 0 {
		if delErr := j.outboxRepo.DeleteBatch(ctx, batchResult.SuccessRecords); delErr != nil {
			slog.ErrorContext(ctx, "ReservationConfirmationOutboxJob: DeleteBatch failed", "err", delErr)
		}
	}

	if len(batchResult.SuccessRecords) > 0 {
		j.metrics.ReportProcessed("success", len(batchResult.SuccessRecords))
	}
	if failedCount > 0 {
		j.metrics.ReportProcessed("failed", failedCount)
	}
	if deadLetterCount > 0 {
		j.metrics.ReportProcessed("dead_letter", deadLetterCount)
	}

	return len(records)
}

func (j *ReservationConfirmationOutboxJob) processBatch(
	ctx context.Context,
	outboxRecords []model.ReservationConfirmationOutboxRecord,
) processBatchResult {
	successRecords := make([]uuid.UUID, 0)
	failedRecordReasons := make(map[uuid.UUID]string)

	for _, outboxRecord := range outboxRecords {
		var data outboxContracts.ReservationConfirmationData
		if err := json.Unmarshal(outboxRecord.Data, &data); err != nil {
			failedRecordReasons[outboxRecord.RecordId] = err.Error()
			continue
		}

		var headers map[string]string
		if err := json.Unmarshal(outboxRecord.Headers, &headers); err != nil {
			failedRecordReasons[outboxRecord.RecordId] = err.Error()
			continue
		}

		recordCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(headers))
		recordCtx, span := j.tracer.Start(recordCtx, "outbox.ConfirmReservation")

		if err := j.productsClient.ConfirmReservation(recordCtx, []int64{data.ReservationId}); err != nil {
			span.RecordError(err)
			span.End()
			failedRecordReasons[outboxRecord.RecordId] = err.Error()
			continue
		}
		span.End()

		successRecords = append(successRecords, outboxRecord.RecordId)
	}

	return processBatchResult{
		SuccessRecords:      successRecords,
		FailedRecordReasons: failedRecordReasons,
	}
}

func findRecord(records []model.ReservationConfirmationOutboxRecord, id uuid.UUID) *model.ReservationConfirmationOutboxRecord {
	for i := range records {
		if records[i].RecordId == id {
			return &records[i]
		}
	}
	return nil
}
