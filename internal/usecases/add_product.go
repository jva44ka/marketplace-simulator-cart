package usecases

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jva44ka/marketplace-simulator-cart/internal/model"
)

type AddProductUseCase struct {
	cartItems     CartItemRepository
	localProducts LocalProductRepository
	productClient ProductClient
}

func NewAddProductUseCase(
	cartItems CartItemRepository,
	localProducts LocalProductRepository,
	productClient ProductClient,
) *AddProductUseCase {
	return &AddProductUseCase{
		cartItems:     cartItems,
		localProducts: localProducts,
		productClient: productClient,
	}
}

func (uc *AddProductUseCase) AddProduct(ctx context.Context, userId uuid.UUID, sku uint64, count uint32) error {
	if count < 1 {
		return model.ErrProductsCountMustBeGreaterThanNull
	}

	productInMasterSystem, err := uc.productClient.GetBySku(ctx, sku)
	if err != nil {
		return fmt.Errorf("productClient.GetBySku: %w", err)
	}

	if productInMasterSystem.Count < count {
		return model.ErrInsufficientStock
	}

	existingCartItem, err := uc.cartItems.GetByUserIdAndSku(ctx, userId, sku)
	if err != nil && !errors.Is(err, model.ErrCartItemsNotFound) {
		return fmt.Errorf("cartItemRepository.GetByUserIdAndSku: %w", err)
	}

	if existingCartItem != nil {
		return uc.cartItems.Update(ctx, existingCartItem.Id, model.CartItem{
			Count: existingCartItem.Count + count,
		})
	}

	_, err = uc.localProducts.GetProductBySku(ctx, sku)
	if err != nil {
		if errors.Is(err, model.ErrProductNotFound) {
			_, err = uc.localProducts.AddProduct(ctx, model.Product{
				Sku:   sku,
				Price: productInMasterSystem.Price,
				Name:  productInMasterSystem.Name,
			})
			if err != nil {
				return fmt.Errorf("productRepository.AddProduct: %w", err)
			}
		} else {
			return fmt.Errorf("productRepository.GetProductBySku: %w", err)
		}
	}

	_, err = uc.cartItems.Create(ctx, model.CartItem{
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
