package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	model2 "github.com/jva44ka/ozon-simulator-go-cart/internal/model"
)

type ProductRepositoryMetrics interface {
	ReportRequest(method, status string)
}

type PgxProductRepository struct {
	pool    *pgxpool.Pool
	metrics ProductRepositoryMetrics
}

func NewPgxProductRepository(pool *pgxpool.Pool, metrics ProductRepositoryMetrics) *PgxProductRepository {
	return &PgxProductRepository{pool: pool, metrics: metrics}
}

type ProductRow struct {
	Sku   uint64
	Price float64
	Name  string
}

func (r *PgxProductRepository) GetProductBySku(ctx context.Context, sku uint64) (model2.Product, error) {
	products, err := r.GetProductsBySku(ctx, []uint64{sku})
	if err != nil {
		return model2.Product{}, err
	}
	if len(products) == 0 {
		return model2.Product{}, model2.ErrProductNotFound
	}
	if len(products) > 1 {
		return model2.Product{}, errors.New("more than one products returned from db")
	}

	return products[0], nil
}

func (r *PgxProductRepository) GetProductsBySku(ctx context.Context, skus []uint64) ([]model2.Product, error) {
	const query = `
SELECT sku, price, name
FROM products
WHERE sku = ANY ($1)
ORDER BY sku DESC`

	rows, err := r.pool.Query(ctx, query, skus)
	if err != nil {
		r.metrics.ReportRequest("GetProductsBySku", "error")
		return nil, fmt.Errorf("ProductRepository.GetProductsBySku: %w", err)
	}

	var productRows []ProductRow
	for rows.Next() {
		var productRow ProductRow
		err = rows.Scan(
			&productRow.Sku,
			&productRow.Price,
			&productRow.Name,
		)

		if err != nil {
			r.metrics.ReportRequest("GetProductsBySku", "error")
			return nil, fmt.Errorf("ProductRepository.GetProductsBySku: %w", err)
		}

		productRows = append(productRows, productRow)
	}

	var result []model2.Product

	for _, productRow := range productRows {
		result = append(result, model2.Product{
			Sku:   productRow.Sku,
			Price: productRow.Price,
			Name:  productRow.Name,
		})
	}

	defer rows.Close()

	r.metrics.ReportRequest("GetProductsBySku", "success")
	return result, nil
}

func (r *PgxProductRepository) AddProduct(ctx context.Context, product model2.Product) (*model2.Product, error) {
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
		r.metrics.ReportRequest("AddProduct", "error")
		return nil, fmt.Errorf("failed to insert products: %w", err)
	}

	r.metrics.ReportRequest("AddProduct", "success")
	return &product, nil
}
