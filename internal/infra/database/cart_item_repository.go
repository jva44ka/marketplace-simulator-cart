package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jva44ka/marketplace-simulator-cart/internal/model"
)

type CartItemRepositoryMetrics interface {
	ReportRequest(method, status string, duration time.Duration)
}

type PgxCartItemRepository struct {
	pool    *pgxpool.Pool
	metrics CartItemRepositoryMetrics
}

func NewPgxCartItemRepository(pool *pgxpool.Pool, metrics CartItemRepositoryMetrics) *PgxCartItemRepository {
	return &PgxCartItemRepository{pool: pool, metrics: metrics}
}

// WithTx returns a transaction-bound view of this repository.
func (r *PgxCartItemRepository) WithTx(tx pgx.Tx) *PgxCartItemTxRepository {
	return &PgxCartItemTxRepository{tx: tx, metrics: r.metrics}
}

// PgxCartItemTxRepository executes cart-item writes inside an open transaction.
type PgxCartItemTxRepository struct {
	tx      pgx.Tx
	metrics CartItemRepositoryMetrics
}

func (r *PgxCartItemTxRepository) RemoveByUserId(ctx context.Context, userId uuid.UUID) error {
	const query = `DELETE FROM cart_items WHERE user_id = $1`

	start := time.Now()
	_, err := r.tx.Exec(ctx, query, userId)
	if err != nil {
		r.metrics.ReportRequest("RemoveByUserId", "error", time.Since(start))
		return fmt.Errorf("PgxCartItemTxRepository.RemoveByUserId: %w", err)
	}

	r.metrics.ReportRequest("RemoveByUserId", "success", time.Since(start))
	return nil
}

// ── non-transactional methods ──────────────────────────────────────────────

type cartItemRow struct {
	id           uint64
	userId       uuid.UUID
	count        uint32
	productSku   uint64
	productPrice float64
	productName  string
}

func (r *PgxCartItemRepository) GetByUserId(ctx context.Context, userId uuid.UUID) ([]model.CartItem, error) {
	const query = `
SELECT
    ci.id,
    ci.user_id,
    ci.count,
    p.sku,
    p.price,
    p.name
FROM cart_items ci
INNER JOIN products p ON p.sku = ci.sku_id
WHERE ci.user_id = $1
ORDER BY ci.id DESC`

	start := time.Now()
	rows, err := r.pool.Query(ctx, query, userId)
	if err != nil {
		r.metrics.ReportRequest("GetByUserId", "error", time.Since(start))
		return nil, fmt.Errorf("PgxCartItemRepository.GetByUserId: %w", err)
	}
	defer rows.Close()

	var result []model.CartItem
	for rows.Next() {
		var row cartItemRow
		if err = rows.Scan(
			&row.id,
			&row.userId,
			&row.count,
			&row.productSku,
			&row.productPrice,
			&row.productName,
		); err != nil {
			r.metrics.ReportRequest("GetByUserId", "error", time.Since(start))
			return nil, fmt.Errorf("PgxCartItemRepository.GetByUserId: %w", err)
		}
		result = append(result, model.CartItem{
			Id:     row.id,
			UserId: row.userId,
			Count:  row.count,
			Product: model.Product{
				Sku:   row.productSku,
				Name:  row.productName,
				Price: row.productPrice,
			},
		})
	}

	r.metrics.ReportRequest("GetByUserId", "success", time.Since(start))
	return result, nil
}

func (r *PgxCartItemRepository) GetByUserIdAndSku(ctx context.Context, userId uuid.UUID, sku uint64) (*model.CartItem, error) {
	const query = `
SELECT
    ci.id,
    ci.user_id,
    ci.count,
    p.sku,
    p.price,
    p.name
FROM cart_items ci
INNER JOIN products p ON p.sku = ci.sku_id
WHERE ci.user_id = $1 AND ci.sku_id = $2`

	start := time.Now()
	var row cartItemRow
	err := r.pool.QueryRow(ctx, query, userId, sku).Scan(
		&row.id,
		&row.userId,
		&row.count,
		&row.productSku,
		&row.productPrice,
		&row.productName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrCartItemsNotFound
		}
		r.metrics.ReportRequest("GetByUserIdAndSku", "error", time.Since(start))
		return nil, fmt.Errorf("PgxCartItemRepository.GetByUserIdAndSku: %w", err)
	}

	r.metrics.ReportRequest("GetByUserIdAndSku", "success", time.Since(start))
	return &model.CartItem{
		Id:     row.id,
		UserId: row.userId,
		Count:  row.count,
		Product: model.Product{
			Sku:   row.productSku,
			Name:  row.productName,
			Price: row.productPrice,
		},
	}, nil
}

func (r *PgxCartItemRepository) Create(ctx context.Context, cartItem model.CartItem) (uint64, error) {
	const query = `
INSERT INTO cart_items (sku_id, user_id, count)
VALUES ($1, $2, $3)
RETURNING id`

	start := time.Now()
	var id int64
	err := r.pool.QueryRow(ctx, query,
		cartItem.Product.Sku,
		cartItem.UserId,
		cartItem.Count,
	).Scan(&id)
	if err != nil {
		r.metrics.ReportRequest("Create", "error", time.Since(start))
		return 0, fmt.Errorf("PgxCartItemRepository.Create: %w", err)
	}

	r.metrics.ReportRequest("Create", "success", time.Since(start))
	return uint64(id), nil
}

func (r *PgxCartItemRepository) Update(ctx context.Context, id uint64, cartItem model.CartItem) error {
	const query = `
UPDATE cart_items
SET count = $2
WHERE id = $1`

	start := time.Now()
	_, err := r.pool.Exec(ctx, query, int64(id), cartItem.Count)
	if err != nil {
		r.metrics.ReportRequest("Update", "error", time.Since(start))
		return fmt.Errorf("PgxCartItemRepository.Update: %w", err)
	}

	r.metrics.ReportRequest("Update", "success", time.Since(start))
	return nil
}

func (r *PgxCartItemRepository) RemoveByUserIdAndSku(ctx context.Context, userId uuid.UUID, sku uint64) error {
	const query = `DELETE FROM cart_items WHERE user_id = $1 AND sku_id = $2`

	start := time.Now()
	_, err := r.pool.Exec(ctx, query, userId, sku)
	if err != nil {
		r.metrics.ReportRequest("RemoveByUserIdAndSku", "error", time.Since(start))
		return fmt.Errorf("PgxCartItemRepository.RemoveByUserIdAndSku: %w", err)
	}

	r.metrics.ReportRequest("RemoveByUserIdAndSku", "success", time.Since(start))
	return nil
}

func (r *PgxCartItemRepository) RemoveByUserId(ctx context.Context, userId uuid.UUID) error {
	const query = `DELETE FROM cart_items WHERE user_id = $1`

	start := time.Now()
	_, err := r.pool.Exec(ctx, query, userId)
	if err != nil {
		r.metrics.ReportRequest("RemoveByUserId", "error", time.Since(start))
		return fmt.Errorf("PgxCartItemRepository.RemoveByUserId: %w", err)
	}

	r.metrics.ReportRequest("RemoveByUserId", "success", time.Since(start))
	return nil
}

func (r *PgxCartItemRepository) CountActiveCarts(ctx context.Context) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(DISTINCT user_id) FROM cart_items").Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("PgxCartItemRepository.CountActiveCarts: %w", err)
	}
	return n, nil
}

func (r *PgxCartItemRepository) CountCartItems(ctx context.Context) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM cart_items").Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("PgxCartItemRepository.CountCartItems: %w", err)
	}
	return n, nil
}
