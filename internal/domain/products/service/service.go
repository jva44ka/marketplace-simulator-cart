package service

import (
	"context"
	"fmt"

	"github.com/jva44ka/ozon-simulator-go-cart/internal/domain/model"
)

type ProductRepository interface {
	GetProductsBySku(ctx context.Context, skus []uint64) ([]model.Product, error)
}

type ProductService struct {
	productRepository ProductRepository
}

func NewProductService(productRepository ProductRepository) *ProductService {
	return &ProductService{productRepository: productRepository}
}

func (s *ProductService) GetProductsBySku(ctx context.Context, skus []uint64) ([]model.Product, error) {
	if len(skus) == 0 {
		return []model.Product{}, nil
	}

	products, err := s.productRepository.GetProductsBySku(ctx, skus)
	if err != nil {
		return nil, fmt.Errorf("ProductRepository.GetProductsBySku :%w", err)
	}

	return products, nil
}
