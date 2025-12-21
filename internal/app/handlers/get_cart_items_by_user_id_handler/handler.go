package get_cart_items_by_user_id_handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/domain/model"
	httpPkg "github.com/jva44ka/ozon-simulator-go-cart/pkg/http"
)

type CartService interface {
	GetItemsByUserId(ctx context.Context, userId uuid.UUID) ([]model.CartItem, error)
}

type ProductService interface {
	GetProductsBySku(ctx context.Context, skus []uint64) ([]model.Product, error)
}

type GetReviewsBySkuHandler struct {
	cartService    CartService
	productService ProductService
}

func NewGetCartItemsByUserIdHandler(cartService CartService, productService ProductService) *GetReviewsBySkuHandler {
	return &GetReviewsBySkuHandler{cartService: cartService, productService: productService}
}

// @Summary      Получить содержимое корзины
// @Description  Метод возвращает содержимое корзины пользователя на текущий момент.
// Если корзины у переданного пользователя нет, либо она пуста, следует вернуть 404 код ответа.
// Товары в корзине упорядочены в порядке возрастания sku.
// @Tags         cart
// @Accept       json
// @Produce      json
// @Param        user_id  path  string  true  "Токен пользователя"
// @Success      200  {object}  CartItemResponse
// @Failure      404  {object}  httpPkg.ErrorResponse
// @Router       /user/{user_id}/cart [get]
func (h *GetReviewsBySkuHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userIdRaw := r.PathValue("user_id")
	userId, err := uuid.Parse(userIdRaw)
	if err != nil {
		if err = httpPkg.NewErrorResponse(w, http.StatusBadRequest, "user_id must be valid uuid"); err != nil {
			fmt.Println("json.Encode failed ", err)

			return
		}

		return
	}

	if userId == uuid.Nil {
		if err = httpPkg.NewErrorResponse(w, http.StatusBadRequest, "userId must be not Nil"); err != nil {
			fmt.Println("json.Encode failed ", err)

			return
		}

		return
	}

	cartItems, err := h.cartService.GetItemsByUserId(r.Context(), userId)
	if err != nil {
		if err = httpPkg.NewErrorResponse(w, http.StatusInternalServerError, err.Error()); err != nil {
			return
		}

		return
	}

	var cartItemsSkus []uint64
	for _, cartItem := range cartItems {
		cartItemsSkus = append(cartItemsSkus, cartItem.SkuId)
	}

	products, err := h.productService.GetProductsBySku(r.Context(), cartItemsSkus)
	if err != nil {
		if err = httpPkg.NewErrorResponse(w, http.StatusInternalServerError, err.Error()); err != nil {
			return
		}

		return
	}

	response := GetReviewsResponse{CartItems: make([]CartItemResponse, 0, len(cartItems))}
	for _, cartItem := range cartItems {
		response.CartItems = append(response.CartItems, CartItemResponse{
			Id:     cartItem.Id,
			SkuId:  cartItem.SkuId,
			UserId: cartItem.UserId,
			Count:  cartItem.Count,
		})
	}

	w.Header().Add("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&response); err != nil {
		fmt.Println("success status failed")
		return
	}

	return
}
