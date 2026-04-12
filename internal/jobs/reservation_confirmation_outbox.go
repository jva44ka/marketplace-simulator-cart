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

type processBatchResult struct {
	SuccessRecords      []uuid.UUID
	FailedRecordReasons map[uuid.UUID]string
}

type ReservationConfirmationOutboxJob struct {
	pool           *pgxpool.Pool
	outboxRepo     OutboxRepository
	productsClient ProductsClient
	enabled        bool
	interval       time.Duration
	batchSize      int
	maxRetryCount  int
}

func NewReservationConfirmationOutboxJob(
	pool *pgxpool.Pool,
	outboxRepo OutboxRepository,
	productsClient ProductsClient,
	enabled bool,
	interval time.Duration,
	batchSize int,
	maxRetryCount int,
) *ReservationConfirmationOutboxJob {
	return &ReservationConfirmationOutboxJob{
		pool:           pool,
		outboxRepo:     outboxRepo,
		productsClient: productsClient,
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
	records, err := j.outboxRepo.GetPending(ctx, j.batchSize)
	if err != nil {
		slog.ErrorContext(ctx, "ReservationConfirmationOutboxJob: GetPending failed", "err", err)
		return
	}

	if len(records) == 0 {
		return
	}

	batchResult := j.processBatch(ctx, records)

	for id, reason := range batchResult.FailedRecordReasons {
		rec := findRecord(records, id)
		if rec == nil {
			continue
		}

		if rec.RetryCount+1 >= j.maxRetryCount {
			if dlErr := j.outboxRepo.MarkDeadLetter(ctx, id, reason); dlErr != nil {
				slog.ErrorContext(ctx, "ReservationConfirmationOutboxJob: MarkDeadLetter failed", "id", id, "err", dlErr)
			}
		} else {
			if retryErr := j.outboxRepo.IncrementRetry(ctx, id); retryErr != nil {
				slog.ErrorContext(ctx, "ReservationConfirmationOutboxJob: IncrementRetry failed", "id", id, "err", retryErr)
			}
		}
	}

	if len(batchResult.SuccessRecords) > 0 {
		successIds := make([]uuid.UUID, 0, len(batchResult.SuccessRecords))
		for _, recordId := range batchResult.SuccessRecords {
			successIds = append(successIds, recordId)
		}

		if delErr := j.outboxRepo.DeleteBatch(ctx, successIds); delErr != nil {
			slog.ErrorContext(ctx, "ReservationConfirmationOutboxJob: DeleteBatch failed", "err", delErr)
		}
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
		if err := j.productsClient.ConfirmReservation(ctx, []int64{data.ReservationId}); err != nil {
			failedRecordReasons[outboxRecord.RecordId] = err.Error()
			continue
		} else {
			successRecords = append(successRecords, outboxRecord.RecordId)
		}
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
