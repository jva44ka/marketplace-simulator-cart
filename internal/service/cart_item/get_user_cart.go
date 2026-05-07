package cart_item

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jva44ka/marketplace-simulator-cart/internal/model"
)

func (s *CartItemService) GetUserCart(ctx context.Context, userId uuid.UUID) ([]model.CartItem, float64, error) {
	cartItems, err := s.cartItems.GetByUserId(ctx, userId)
	if err != nil {
		return nil, 0.0, fmt.Errorf("cartItems.GetByUserId: %w", err)
	}

	totalPrice := 0.0
	for _, cartItem := range cartItems {
		totalPrice += cartItem.Product.Price * float64(cartItem.Count)
	}

	return cartItems, totalPrice, nil
}
