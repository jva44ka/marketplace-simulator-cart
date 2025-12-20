package model

import "errors"

var (
	ErrProductNotFound   = errors.New("product not found")
	ErrCartItemsNotFound = errors.New("cartItems not found")
	ErrProductsNotFound  = errors.New("products not found")
)
