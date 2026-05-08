package usecases

import (
	"context"

	"github.com/google/uuid"
)

type RemoveAllProductsUseCase struct {
	cartItems CartItemRepository
}

func NewRemoveAllProductsUseCase(cartItems CartItemRepository) *RemoveAllProductsUseCase {
	return &RemoveAllProductsUseCase{cartItems: cartItems}
}

func (uc *RemoveAllProductsUseCase) Execute(ctx context.Context, userId uuid.UUID) error {
	return uc.cartItems.RemoveByUserId(ctx, userId)
}
