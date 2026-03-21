package validation

import (
	"strconv"

	"github.com/google/uuid"
)

type Validator struct{}

func (v Validator) GetValidatedSku(skuRaw string) (uint64, error) {
	sku, err := strconv.Atoi(skuRaw)
	if err != nil {
		return 0, NewValidationError("sku must be a number")
	}

	if sku < 1 {
		return 0, NewValidationError("sku must be more than zero")
	}

	return uint64(sku), nil
}

func (v Validator) GetValidatedUserId(userIdRaw string) (uuid.UUID, error) {
	userId, err := uuid.Parse(userIdRaw)
	if err != nil {
		return uuid.Nil, NewValidationError("user_id must be valid uuid")
	}

	if userId == uuid.Nil {
		return uuid.Nil, NewValidationError("user_id must be not nil")
	}

	return userId, nil
}
