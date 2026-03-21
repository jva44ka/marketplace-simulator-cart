package get_cart_items_by_user_id_handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/domain/model"
	httpPkg "github.com/jva44ka/ozon-simulator-go-cart/pkg/http"
)

type CartService interface {
	GetItemsByUserId(ctx context.Context, userId uuid.UUID) ([]model.CartItem, float64, error)
}

type Validator interface {
	GetValidatedSku(skuRaw string) (uint64, error)
	GetValidatedUserId(userIdRaw string) (uuid.UUID, error)
}

type GetReviewsBySkuHandler struct {
	cartService CartService
	validator   Validator
}

func NewGetCartItemsByUserIdHandler(cartService CartService, validator Validator) *GetReviewsBySkuHandler {
	return &GetReviewsBySkuHandler{
		cartService: cartService,
		validator:   validator,
	}
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
	userId, err := h.validator.GetValidatedUserId(userIdRaw)
	if err != nil {
		httpPkg.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	cartItems, totalPrice, err := h.cartService.GetItemsByUserId(r.Context(), userId)
	if err != nil {
		httpPkg.WriteServiceError(w, err)
		return
	}

	response := GetReviewsResponse{
		CartItems:  make([]CartItemResponse, 0, len(cartItems)),
		TotalPrice: totalPrice,
	}
	for _, cartItem := range cartItems {
		response.CartItems = append(response.CartItems, CartItemResponse{
			Id:    cartItem.Id,
			Sku:   cartItem.Product.Sku,
			Name:  cartItem.Product.Name,
			Price: cartItem.Product.Price,
			Count: cartItem.Count,
		})
	}

	httpPkg.WriteSuccessResponse(w, response)
	return
}
