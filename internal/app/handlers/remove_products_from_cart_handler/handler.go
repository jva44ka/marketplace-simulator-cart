package remove_products_from_cart_handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	httpPkg "github.com/jva44ka/ozon-simulator-go-cart/pkg/http"
)

type CartService interface {
	RemoveProduct(ctx context.Context, userId uuid.UUID, sku uint64) error
}

type Validator interface {
	GetValidatedSku(skuRaw string) (uint64, error)
	GetValidatedUserId(userIdRaw string) (uuid.UUID, error)
}

type RemoveProductsFromCartHandler struct {
	cartService CartService
	validator   Validator
}

func NewRemoveProductsFromCartHandler(cartService CartService, validator Validator) *RemoveProductsFromCartHandler {
	return &RemoveProductsFromCartHandler{
		cartService: cartService,
		validator:   validator,
	}
}

// @Summary      Удалить товар из корзины
// @Description  Метод полностью удаляет все количество товара из корзины пользователя.
// Если у пользователя вовсе нет данной позиции, то возвращается такой же ответ, как будто бы все позиции данного sku были успешно удалены
// @Tags         cart
// @Accept       json
// @Produce      json
// @Param        user_id  path  string  true  "Токен пользователя"
// @Param        sku   path  uint64  true  "SKU товара"
// @Success      200  {object}  RemoveProductsFromCartResponse
// @Failure      404  {object}  httpPkg.ErrorResponse
// @Router       /user/{user_id}/cart/{sku} [delete]
func (h *RemoveProductsFromCartHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	skuRaw := r.PathValue("sku")
	sku, err := h.validator.GetValidatedSku(skuRaw)
	if err != nil {
		httpPkg.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	userIdRaw := r.PathValue("user_id")
	userId, err := h.validator.GetValidatedUserId(userIdRaw)
	if err != nil {
		httpPkg.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	err = h.cartService.RemoveProduct(r.Context(), userId, uint64(sku))
	if err != nil {
		httpPkg.WriteServiceError(w, err)
		return
	}

	httpPkg.WriteSuccessEmptyResponse(w)

	return
}
