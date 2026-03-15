package get_cart_items_by_user_id_handler

type GetReviewsResponse struct {
	CartItems  []CartItemResponse `json:"cart_items"`
	TotalPrice float64            `json:"total_price"`
}

type CartItemResponse struct {
	Id    uint64  `json:"id"`
	Sku   uint64  `json:"sku"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Count uint32  `json:"count"`
}
