package model

import "errors"

var (
	ErrProductNotFound                    = errors.New("products not found")
	ErrProductsCountMustBeGreaterThanNull = errors.New("count must be greater than zero")
	ErrCartItemsNotFound                  = errors.New("cart items not found")
	ErrCartEmpty                          = errors.New("cart is empty")
	ErrInsufficientStock                  = errors.New("insufficient stock")
)
