package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/domain/model"
)

type PgxProductRepository struct {
	pool *pgxpool.Pool
}

func NewPgxProductRepository(pool *pgxpool.Pool) *PgxProductRepository {
	return &PgxProductRepository{pool: pool}
}

type ProductRow struct {
	Sku   uint64
	Price float64
	Name  string
}

func (r *PgxProductRepository) GetProductBySku(ctx context.Context, sku uint64) (model.Product, error) {
	products, err := r.GetProductsBySku(ctx, []uint64{sku})
	if err != nil {
		return model.Product{}, err
	}
	if len(products) == 0 {
		return model.Product{}, errors.New("zero products returned from db")
	}
	if len(products) > 1 {
		return model.Product{}, errors.New("more than one products returned from db")
	}

	return products[0], nil
}

func (r *PgxProductRepository) GetProductsBySku(ctx context.Context, skus []uint64) ([]model.Product, error) {
	const query = `
SELECT sku, price, name
FROM products 
WHERE sku IN ($1)
ORDER BY sku DESC`

	rows, err := r.pool.Query(ctx, query, skus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrProductsNotFound
		}
	}

	var ProductRows []ProductRow
	for rows.Next() {
		var ProductRow ProductRow
		err = rows.Scan(
			&ProductRow.Sku,
			&ProductRow.Price,
			&ProductRow.Name)

		if err != nil {
			return nil, fmt.Errorf("ProductRepository.GetProductsBySku: %w", err)
		}

		ProductRows = append(ProductRows, ProductRow)
	}

	var result []model.Product

	for _, ProductRow := range ProductRows {
		result = append(result, model.Product{
			Sku:   ProductRow.Sku,
			Price: ProductRow.Price,
			Name:  ProductRow.Name,
		})
	}

	defer rows.Close()

	return result, nil
}

func (r *PgxProductRepository) AddProduct(ctx context.Context, product model.Product) (*model.Product, error) {
	const query = `
INSERT INTO 
    products (sku, price, name) 
VALUES 
    ($1, $2, $3);`

	err := pgx.BeginTxFunc(ctx, r.pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, query, product.Sku, product.Price, product.Name)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to insert product: %w", err)
	}

	return &product, nil
}
