package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/model"
	cartItemPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/service/cart_item"
)

type ReservationConfirmationOutboxPgxRepository struct {
	pool *pgxpool.Pool
}

func NewReservationConfirmationOutboxPgxRepository(pool *pgxpool.Pool) *ReservationConfirmationOutboxPgxRepository {
	return &ReservationConfirmationOutboxPgxRepository{pool: pool}
}

func (r *ReservationConfirmationOutboxPgxRepository) GetPending(ctx context.Context, limit int) ([]model.ReservationConfirmationOutboxRecord, error) {
	const query = `
SELECT DISTINCT ON (key)
    record_id,
    key,
    data,
    headers,
    created_at,
    retry_count,
    is_dead_letter,
    marked_as_dead_letter_at,
    dead_letter_reason
FROM outbox.reservation_confirmation_events
WHERE is_dead_letter = FALSE
ORDER BY key, created_at
LIMIT $1`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("ReservationConfirmationOutboxPgxRepository.GetPending: %w", err)
	}
	defer rows.Close()

	var result []model.ReservationConfirmationOutboxRecord
	for rows.Next() {
		var rec model.ReservationConfirmationOutboxRecord
		if err = rows.Scan(
			&rec.RecordId,
			&rec.Key,
			&rec.Data,
			&rec.Headers,
			&rec.CreatedAt,
			&rec.RetryCount,
			&rec.IsDeadLetter,
			&rec.MarkedAsDeadLetterAt,
			&rec.DeadLetterReason,
		); err != nil {
			return nil, fmt.Errorf("ReservationConfirmationOutboxPgxRepository.GetPending scan: %w", err)
		}
		result = append(result, rec)
	}

	return result, nil
}

func (r *ReservationConfirmationOutboxPgxRepository) DeleteBatch(ctx context.Context, ids []uuid.UUID) error {
	const query = `DELETE FROM outbox.reservation_confirmation_events WHERE record_id = ANY($1)`

	_, err := r.pool.Exec(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("ReservationConfirmationOutboxPgxRepository.DeleteBatch: %w", err)
	}

	return nil
}

func (r *ReservationConfirmationOutboxPgxRepository) IncrementRetry(ctx context.Context, id uuid.UUID) error {
	const query = `UPDATE outbox.reservation_confirmation_events SET retry_count = retry_count + 1 WHERE record_id = $1`

	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("ReservationConfirmationOutboxPgxRepository.IncrementRetry: %w", err)
	}

	return nil
}

func (r *ReservationConfirmationOutboxPgxRepository) MarkDeadLetter(ctx context.Context, id uuid.UUID, reason string) error {
	const query = `
UPDATE outbox.reservation_confirmation_events
SET is_dead_letter = TRUE, marked_as_dead_letter_at = $2, dead_letter_reason = $3
WHERE record_id = $1`

	now := time.Now()
	_, err := r.pool.Exec(ctx, query, id, now, reason)
	if err != nil {
		return fmt.Errorf("ReservationConfirmationOutboxPgxRepository.MarkDeadLetter: %w", err)
	}

	return nil
}

func (r *ReservationConfirmationOutboxPgxRepository) WithTx(tx pgx.Tx) cartItemPkg.OutboxTxRepository {
	return &ReservationConfirmationOutboxPgxTxRepository{tx: tx}
}

// ReservationConfirmationOutboxPgxTxRepository is the transactional version used during checkout.
type ReservationConfirmationOutboxPgxTxRepository struct {
	tx pgx.Tx
}

func (r *ReservationConfirmationOutboxPgxTxRepository) Create(ctx context.Context, rec model.ReservationConfirmationOutboxRecordNew) error {
	const query = `
INSERT INTO outbox.reservation_confirmation_events (key, data, headers)
VALUES ($1, $2, $3)`

	_, err := r.tx.Exec(ctx, query, rec.Key, rec.Data, rec.Headers)
	if err != nil {
		return fmt.Errorf("ReservationConfirmationOutboxPgxTxRepository.Create: %w", err)
	}

	return nil
}
