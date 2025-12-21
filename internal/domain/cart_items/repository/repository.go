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
	id           uint64
	userId       uuid.UUID
	count        uint32
	productSku   uint64
	productPrice float64
	productName  string
}

func (r *PgxCartItemRepository) GetCartItemsByUserId(ctx context.Context, userId uuid.UUID) ([]model.CartItem, error) {
	const query = `
SELECT
    ci.id AS id,
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
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrCartItemsNotFound
		}
	}

	var cartItemRows []CartItemRow
	for rows.Next() {
		var cartItemRow CartItemRow
		err = rows.Scan(
			&cartItemRow.id,
			&cartItemRow.userId,
			&cartItemRow.count,
			&cartItemRow.productSku,
			&cartItemRow.productPrice,
			&cartItemRow.productName)

		if err != nil {
			return nil, fmt.Errorf("CartItemRepository.GetCartItemsByUserId: %w", err)
		}

		cartItemRows = append(cartItemRows, cartItemRow)
	}

	var result []model.CartItem

	for _, cartItemRow := range cartItemRows {
		result = append(result, model.CartItem{
			Id:     cartItemRow.id,
			UserId: cartItemRow.userId,
			Count:  cartItemRow.count,
			Product: model.Product{
				Sku:   cartItemRow.productSku,
				Name:  cartItemRow.productName,
				Price: cartItemRow.productPrice,
			},
		})
	}

	defer rows.Close()

	return result, nil
}

func (r *PgxCartItemRepository) GetCartItem(ctx context.Context, userId uuid.UUID, sku uint64) (*model.CartItem, error) {
	const query = `
SELECT
    ci.id AS id,
    ci.user_id,
    ci.count,

    p.sku,
    p.price,
    p.name
FROM cart_items ci
INNER JOIN products p ON p.sku = ci.sku_id
WHERE 
    ci.user_id = $1
    AND ci.sku_id = $2`

	row := r.pool.QueryRow(ctx, query, userId, sku)

	var cartItemRow = CartItemRow{}

	err := row.Scan(
		&cartItemRow.id,
		&cartItemRow.userId,
		&cartItemRow.count,
		&cartItemRow.productSku,
		&cartItemRow.productPrice,
		&cartItemRow.productName)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrCartItemsNotFound
		}
		return nil, fmt.Errorf("PgxCartItemRepository.GetCartItem: %w", err)
	}

	// Преобразуем типы в модель приложения
	result := &model.CartItem{
		Id:     cartItemRow.id,
		UserId: cartItemRow.userId,
		Count:  cartItemRow.count,
		Product: model.Product{
			Sku:   cartItemRow.productSku,
			Name:  cartItemRow.productName,
			Price: cartItemRow.productPrice,
		},
	}

	return result, nil
}

func (r *PgxCartItemRepository) AddCartItem(ctx context.Context, cartItem model.CartItem) (uint64, error) {
	const query = `
INSERT INTO 
    cart_items (sku_id, user_id, count) 
VALUES 
    ($1, $2, $3)
RETURNING 
	id;`

	var id int64
	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, query, cartItem.Product.Sku, cartItem.UserId, cartItem.Count).Scan(&id)
	})
	if err != nil {
		return 0, fmt.Errorf("failed to insert cart item: %w", err)
	}

	return uint64(id), nil
}

func (r *PgxCartItemRepository) UpdateCartItem(ctx context.Context, id uint64, cartItem model.CartItem) error {
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
		return fmt.Errorf("failed to insert cart item: %w", err)
	}

	return nil
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
