package cart_item

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jva44ka/marketplace-simulator-cart/internal/model"
)

func (s *CartItemService) AddProduct(ctx context.Context, userId uuid.UUID, sku uint64, count uint32) error {
	if count < 1 {
		return model.ErrProductsCountMustBeGreaterThanNull
	}

	productInMasterSystem, err := s.productClient.GetBySku(ctx, sku)
	if err != nil {
		return fmt.Errorf("productClient.GetBySku: %w", err)
	}

	if productInMasterSystem.Count < count {
		return model.ErrInsufficientStock
	}

	existingCartItem, err := s.db.CartItemRepo().GetByUserIdAndSku(ctx, userId, sku)
	if err != nil && !errors.Is(err, model.ErrCartItemsNotFound) {
		return fmt.Errorf("cartRepository.GetByUserIdAndSku: %w", err)
	}

	if existingCartItem != nil {
		return s.db.CartItemRepo().Update(ctx, existingCartItem.Id, model.CartItem{
			Count: existingCartItem.Count + count,
		})
	}

	// Убеждаемся что продукт есть в локальной БД
	_, err = s.db.ProductRepo().GetProductBySku(ctx, sku)
	if err != nil {
		if errors.Is(err, model.ErrProductNotFound) {
			_, err = s.db.ProductRepo().AddProduct(ctx, model.Product{
				Sku:   sku,
				Price: productInMasterSystem.Price,
				Name:  productInMasterSystem.Name,
			})
			if err != nil {
				return fmt.Errorf("productRepository.AddProduct: %w", err)
			}
		} else {
			return fmt.Errorf("productRepository.GetProductsBySku: %w", err)
		}
	}

	_, err = s.db.CartItemRepo().Create(ctx, model.CartItem{
		UserId: userId,
		Count:  count,
		Product: model.Product{
			Sku:   sku,
			Price: productInMasterSystem.Price,
			Name:  productInMasterSystem.Name,
		},
	})
	return err
}
