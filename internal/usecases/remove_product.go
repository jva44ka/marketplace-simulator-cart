package usecases

import (
	"context"

	"github.com/google/uuid"
)

type RemoveProductUseCase struct {
	cartItems CartItemRepository
}

func NewRemoveProductUseCase(cartItems CartItemRepository) *RemoveProductUseCase {
	return &RemoveProductUseCase{cartItems: cartItems}
}

func (uc *RemoveProductUseCase) RemoveProduct(ctx context.Context, userId uuid.UUID, sku uint64) error {
	return uc.cartItems.RemoveByUserIdAndSku(ctx, userId, sku)
}
