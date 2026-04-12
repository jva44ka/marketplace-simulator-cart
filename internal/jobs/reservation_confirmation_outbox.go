package jobs

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	outboxContracts "github.com/jva44ka/ozon-simulator-go-cart/api_internal/outbox"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/model"
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
	ReportConfirmationDuration(d time.Duration)
	ReportRecordAge(age time.Duration)
}

type processBatchResult struct {
	SuccessRecords      []uuid.UUID
	FailedRecordReasons map[uuid.UUID]string
}

type ReservationConfirmationOutboxJob struct {
	pool           *pgxpool.Pool
	outboxRepo     OutboxRepository
	productsClient ProductsClient
	metrics        OutboxJobMetrics
	enabled        bool
	interval       time.Duration
	batchSize      int
	maxRetryCount  int
}

func NewReservationConfirmationOutboxJob(
	pool *pgxpool.Pool,
	outboxRepo OutboxRepository,
	productsClient ProductsClient,
	metrics OutboxJobMetrics,
	enabled bool,
	interval time.Duration,
	batchSize int,
	maxRetryCount int,
) *ReservationConfirmationOutboxJob {
	return &ReservationConfirmationOutboxJob{
		pool:           pool,
		outboxRepo:     outboxRepo,
		productsClient: productsClient,
		metrics:        metrics,
		enabled:        enabled,
		interval:       interval,
		batchSize:      batchSize,
		maxRetryCount:  maxRetryCount,
	}
}

func (j *ReservationConfirmationOutboxJob) Run(ctx context.Context) {
	if !j.enabled {
		slog.InfoContext(ctx, "ReservationConfirmationOutboxJob disabled, shutting down")
		return
	}

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.tick(ctx)
		}
	}
}

func (j *ReservationConfirmationOutboxJob) tick(ctx context.Context) {
	tickStart := time.Now()
	defer func() {
		j.metrics.ReportTickDuration(time.Since(tickStart))
	}()

	records, err := j.outboxRepo.GetPending(ctx, j.batchSize)
	if err != nil {
		slog.ErrorContext(ctx, "ReservationConfirmationOutboxJob: GetPending failed", "err", err)
		return
	}

	if len(records) == 0 {
		return
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

		if rec.RetryCount+1 >= j.maxRetryCount {
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

	// Репортим метрики по результатам
	if len(batchResult.SuccessRecords) > 0 {
		j.metrics.ReportProcessed("success", len(batchResult.SuccessRecords))
	}
	if failedCount > 0 {
		j.metrics.ReportProcessed("failed", failedCount)
	}
	if deadLetterCount > 0 {
		j.metrics.ReportProcessed("dead_letter", deadLetterCount)
	}
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

		//TODO: прокидывать заголовки в запрос
		confirmStart := time.Now()
		if err := j.productsClient.ConfirmReservation(ctx, []int64{data.ReservationId}); err != nil {
			j.metrics.ReportConfirmationDuration(time.Since(confirmStart))
			failedRecordReasons[outboxRecord.RecordId] = err.Error()
			continue
		}
		j.metrics.ReportConfirmationDuration(time.Since(confirmStart))

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
