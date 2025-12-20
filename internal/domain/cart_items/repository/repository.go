package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/domain/model"
)

type PgxCartItemRepository struct {
	pool *pgxpool.Pool
}

func NewPgxCartItemRepository(pool *pgxpool.Pool) *PgxCartItemRepository {
	return &PgxCartItemRepository{pool: pool}
}

type CartItemRow struct {
	Id     uint64
	SkuId  uint64
	UserId uuid.UUID
	Count  uint32
}

func (r *PgxCartItemRepository) GetCartItemsByUserId(ctx context.Context, userId uuid.UUID) ([]model.CartItem, error) {
	const query = `
SELECT id, sku_id, user_id, count 
FROM cart_items 
WHERE user_id = $1
ORDER BY id DESC`

	rows, err := r.pool.Query(ctx, query, userId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrCartItemsNotFound
		}
	}

	var cartItemRows []CartItemRow
	for rows.Next() {
		var cartItemRow CartItemRow
		err = rows.Scan(
			&cartItemRow.Id,
			&cartItemRow.SkuId,
			&cartItemRow.UserId,
			&cartItemRow.Count)

		if err != nil {
			return nil, fmt.Errorf("CartItemRepository.GetCartItemsByUserId: %w", err)
		}

		cartItemRows = append(cartItemRows, cartItemRow)
	}

	var result []model.CartItem

	for _, cartItemRow := range cartItemRows {
		result = append(result, model.CartItem{
			Id:     cartItemRow.Id,
			SkuId:  cartItemRow.SkuId,
			UserId: cartItemRow.UserId,
			Count:  cartItemRow.Count,
		})
	}

	defer rows.Close()

	return result, nil
}

func (r *PgxCartItemRepository) GetCartItem(ctx context.Context, userId uuid.UUID, sku uint64) (*model.CartItem, error) {
	const query = `
SELECT 
    id, sku_id, user_id, count 
FROM 
    cart_items 
WHERE 
    user_id = $1
    AND sku_id = $2`

	row := r.pool.QueryRow(ctx, query, userId, sku)

	var productRow = CartItemRow{}

	err := row.Scan(&productRow.Id, &productRow.SkuId, &productRow.UserId, &productRow.Count)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrCartItemsNotFound
		}
		return nil, fmt.Errorf("PgxCartItemRepository.GetCartItem: %w", err)
	}

	// Преобразуем типы в модель приложения
	result := &model.CartItem{
		Id:     productRow.Id,
		SkuId:  productRow.SkuId,
		UserId: productRow.UserId,
		Count:  productRow.Count,
	}

	return result, nil
}

func (r *PgxCartItemRepository) AddCartItem(ctx context.Context, cartItem model.CartItem) (*model.CartItem, error) {
	const query = `
INSERT INTO 
    cart_items (sku_id, user_id, count) 
VALUES 
    ($1, $2, $3)
RETURNING 
	id;`

	var id int64
	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, cartItem.SkuId, cartItem.UserId, cartItem.Count).Scan(&id)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert cart item: %w", err)
	}

	result := model.CartItem{
		Id:     uint64(id),
		SkuId:  cartItem.SkuId,
		UserId: cartItem.UserId,
		Count:  cartItem.Count,
	}

	return &result, nil
}

func (r *PgxCartItemRepository) UpdateCartItem(ctx context.Context, id uint64, cartItem model.CartItem) (*model.CartItem, error) {
	const query = `
UPDATE 
    cart_items
SET
	count = $2
WHERE 
    id = $1`

	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, int64(id), cartItem.Count)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert cart item: %w", err)
	}

	result := model.CartItem{
		Id:     id,
		SkuId:  cartItem.SkuId,
		UserId: cartItem.UserId,
		Count:  cartItem.Count,
	}

	return &result, nil
}

func (r *PgxCartItemRepository) RemoveCartItem(ctx context.Context, userId uuid.UUID, sku uint64) error {
	const query = `
DELETE FROM
    cart_items
WHERE 
    user_id = $1
	AND sku_id = $2;`

	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, userId, sku)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to delete cart item: %w", err)
	}

	return nil
}

func (r *PgxCartItemRepository) RemoveAllCartItemsByUserId(ctx context.Context, userId uuid.UUID) error {
	const query = `
DELETE FROM
    cart_items
WHERE 
    user_id = $1;`

	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, userId)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to delete all cart items by user id: %w", err)
	}

	return nil
}
