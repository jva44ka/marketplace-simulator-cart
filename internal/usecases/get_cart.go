package usecases

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jva44ka/marketplace-simulator-cart/internal/model"
)

type GetCartUseCase struct {
	cartItems CartItemRepository
}

func NewGetCartUseCase(cartItems CartItemRepository) *GetCartUseCase {
	return &GetCartUseCase{cartItems: cartItems}
}

func (uc *GetCartUseCase) GetUserCart(ctx context.Context, userId uuid.UUID) ([]model.CartItem, float64, error) {
	cartItems, err := uc.cartItems.GetByUserId(ctx, userId)
	if err != nil {
		return nil, 0.0, fmt.Errorf("cartItemRepository.GetByUserId: %w", err)
	}

	totalPrice := 0.0
	for _, item := range cartItems {
		totalPrice += item.Product.Price * float64(item.Count)
	}

	return cartItems, totalPrice, nil
}
