package cart_item

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/model"
)

func (s *CartItemService) GetUserCart(ctx context.Context, userId uuid.UUID) ([]model.CartItem, float64, error) {
	cartItems, err := s.cartItemRepository.GetByUserId(ctx, userId)
	if err != nil {
		return nil, 0.0, fmt.Errorf("cartRepository.GetByUserId: %w", err)
	}

	totalPrice := 0.0
	for _, cartItem := range cartItems {
		totalPrice += cartItem.Product.Price
	}

	return cartItems, totalPrice, nil
}
