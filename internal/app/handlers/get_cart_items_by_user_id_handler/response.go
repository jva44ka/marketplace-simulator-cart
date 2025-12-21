package get_cart_items_by_user_id_handler

type GetReviewsResponse struct {
	CartItems []CartItemResponse `json:"cart_items"`
}

type CartItemResponse struct {
	Id          uint64 `json:"id"`
	SkuId       uint64 `json:"sku_id"`
	ProductName string `json:"product_name"`
	Count       uint32 `json:"count"`
}
