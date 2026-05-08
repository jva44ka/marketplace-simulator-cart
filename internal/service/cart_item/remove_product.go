package cart_item

import (
	"context"

	"github.com/google/uuid"
)

func (s *CartItemService) RemoveProduct(ctx context.Context, userId uuid.UUID, sku uint64) error {
	return s.cartItemRepository.RemoveByUserIdAndSku(ctx, userId, sku)
}
