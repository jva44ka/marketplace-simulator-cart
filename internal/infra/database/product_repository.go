package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jva44ka/marketplace-simulator-cart/internal/model"
)


type ProductRepositoryMetrics interface {
	ReportRequest(method, status string, duration time.Duration)
}

type PgxProductRepository struct {
	pool    *pgxpool.Pool
	metrics ProductRepositoryMetrics
}

func NewPgxProductRepository(pool *pgxpool.Pool, metrics ProductRepositoryMetrics) *PgxProductRepository {
	return &PgxProductRepository{pool: pool, metrics: metrics}
}

func (r *PgxProductRepository) GetProductBySku(ctx context.Context, sku uint64) (model.Product, error) {
	products, err := r.getProductsBySku(ctx, []uint64{sku})
	if err != nil {
		return model.Product{}, err
	}
	if len(products) == 0 {
		return model.Product{}, model.ErrProductNotFound
	}
	if len(products) > 1 {
		return model.Product{}, errors.New("more than one product returned from db")
	}
	return products[0], nil
}

func (r *PgxProductRepository) getProductsBySku(ctx context.Context, skus []uint64) ([]model.Product, error) {
	const query = `
SELECT sku, price, name
FROM products
WHERE sku = ANY ($1)
ORDER BY sku DESC`

	start := time.Now()
	rows, err := r.pool.Query(ctx, query, skus)
	if err != nil {
		r.metrics.ReportRequest("GetProductsBySku", "error", time.Since(start))
		return nil, fmt.Errorf("PgxProductRepository.GetProductsBySku: %w", err)
	}
	defer rows.Close()

	var result []model.Product
	for rows.Next() {
		var p model.Product
		if err = rows.Scan(&p.Sku, &p.Price, &p.Name); err != nil {
			r.metrics.ReportRequest("GetProductsBySku", "error", time.Since(start))
			return nil, fmt.Errorf("PgxProductRepository.GetProductsBySku: %w", err)
		}
		result = append(result, p)
	}

	r.metrics.ReportRequest("GetProductsBySku", "success", time.Since(start))
	return result, nil
}

func (r *PgxProductRepository) AddProduct(ctx context.Context, product model.Product) (*model.Product, error) {
	const query = `
INSERT INTO products (sku, price, name)
VALUES ($1, $2, $3)`

	start := time.Now()
	_, err := r.pool.Exec(ctx, query, product.Sku, product.Price, product.Name)
	if err != nil {
		r.metrics.ReportRequest("AddProduct", "error", time.Since(start))
		return nil, fmt.Errorf("PgxProductRepository.AddProduct: %w", err)
	}

	r.metrics.ReportRequest("AddProduct", "success", time.Since(start))
	return &product, nil
}
