package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/model"
	cartItemPkg "github.com/jva44ka/ozon-simulator-go-cart/internal/service/cart_item"
)

type CartItemRepositoryMetrics interface {
	ReportRequest(method, status string)
}

type PgxCartItemRepository struct {
	pool    *pgxpool.Pool
	metrics CartItemRepositoryMetrics
}

func NewPgxCartItemRepository(pool *pgxpool.Pool, metrics CartItemRepositoryMetrics) *PgxCartItemRepository {
	return &PgxCartItemRepository{pool: pool, metrics: metrics}
}

type CartItemRow struct {
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

	rows, err := r.pool.Query(ctx, query, userId)
	if err != nil {
		return nil, fmt.Errorf("PgxCartItemRepository.GetByUserId: %w", err)
	}
	defer rows.Close()

	var result []model.CartItem
	for rows.Next() {
		var row CartItemRow
		if err = rows.Scan(
			&row.id,
			&row.userId,
			&row.count,
			&row.productSku,
			&row.productPrice,
			&row.productName,
		); err != nil {
			r.metrics.ReportRequest("GetByUserId", "error")
			return nil, fmt.Errorf("CartItemRepository.GetByUserId: %w", err)
		}

		item := model.CartItem{
			Id:     row.id,
			UserId: row.userId,
			Count:  row.count,
			Product: model.Product{
				Sku:   row.productSku,
				Name:  row.productName,
				Price: row.productPrice,
			},
		}

		result = append(result, item)
	}

	r.metrics.ReportRequest("GetByUserId", "success")
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

	row := r.pool.QueryRow(ctx, query, userId, sku)

	var cr CartItemRow
	err := row.Scan(
		&cr.id,
		&cr.userId,
		&cr.count,
		&cr.productSku,
		&cr.productPrice,
		&cr.productName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrCartItemsNotFound
		}
		r.metrics.ReportRequest("GetByUserIdAndSku", "error")
		return nil, fmt.Errorf("PgxCartItemRepository.GetByUserIdAndSku: %w", err)
	}

	r.metrics.ReportRequest("GetByUserIdAndSku", "success")
	item := &model.CartItem{
		Id:     cr.id,
		UserId: cr.userId,
		Count:  cr.count,
		Product: model.Product{
			Sku:   cr.productSku,
			Name:  cr.productName,
			Price: cr.productPrice,
		},
	}

	return item, nil
}


func (r *PgxCartItemRepository) Create(ctx context.Context, cartItem model.CartItem) (uint64, error) {
	const query = `
INSERT INTO cart_items (sku_id, user_id, count)
VALUES ($1, $2, $3)
RETURNING id`

	var id int64
	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query,
			cartItem.Product.Sku,
			cartItem.UserId,
			cartItem.Count,
		).Scan(&id)
	})
	if err != nil {
		r.metrics.ReportRequest("Create", "error")
		return 0, fmt.Errorf("failed to insert cart item: %w", err)
	}

	r.metrics.ReportRequest("Create", "success")
	return uint64(id), nil
}

func (r *PgxCartItemRepository) Update(ctx context.Context, id uint64, cartItem model.CartItem) error {
	const query = `
UPDATE cart_items
SET 
    count = $2
WHERE id = $1`

	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, int64(id), cartItem.Count)
		return err
	})
	if err != nil {
		r.metrics.ReportRequest("Update", "error")
		return fmt.Errorf("failed to update cart item: %w", err)
	}

	r.metrics.ReportRequest("Update", "success")
	return nil
}

func (r *PgxCartItemRepository) RemoveByUserIdAndSku(ctx context.Context, userId uuid.UUID, sku uint64) error {
	const query = `DELETE FROM cart_items WHERE user_id = $1 AND sku_id = $2`

	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, userId, sku)
		return err
	})
	if err != nil {
		r.metrics.ReportRequest("RemoveByUserIdAndSku", "error")
		return fmt.Errorf("failed to delete cart item: %w", err)
	}

	r.metrics.ReportRequest("RemoveByUserIdAndSku", "success")
	return nil
}

func (r *PgxCartItemRepository) RemoveByUserId(ctx context.Context, userId uuid.UUID) error {
	const query = `DELETE FROM cart_items WHERE user_id = $1`

	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, userId)
		return err
	})
	if err != nil {
		r.metrics.ReportRequest("RemoveByUserId", "error")
		return fmt.Errorf("failed to delete all cart items by user id: %w", err)
	}

	r.metrics.ReportRequest("RemoveByUserId", "success")
	return nil
}

func (r *PgxCartItemRepository) WithTx(tx pgx.Tx) cartItemPkg.CartItemTxRepository {
	return &PgxCartItemTxRepository{tx: tx}
}

type PgxCartItemTxRepository struct {
	tx pgx.Tx
}

func (r *PgxCartItemTxRepository) RemoveByUserId(ctx context.Context, userId uuid.UUID) error {
	const query = `DELETE FROM cart_items WHERE user_id = $1`

	_, err := r.tx.Exec(ctx, query, userId)
	if err != nil {
		return fmt.Errorf("PgxCartItemTxRepository.RemoveByUserId: %w", err)
	}

	return nil
}
