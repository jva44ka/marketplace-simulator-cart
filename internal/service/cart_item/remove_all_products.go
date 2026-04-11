package cart_item

import (
	"context"

	"github.com/google/uuid"
)

func (s *CartItemService) RemoveAllProducts(ctx context.Context, userId uuid.UUID) error {
	return s.db.CartItemRepo().RemoveByUserId(ctx, userId)
}
