package remove_products_from_cart_handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	httpPkg "github.com/jva44ka/ozon-simulator-go-cart/pkg/http"
)

type CartService interface {
	RemoveProduct(ctx context.Context, userId uuid.UUID, sku uint64) error
}

type RemoveProductsFromCartHandler struct {
	cartService CartService
}

func NewRemoveProductsFromCartHandler(cartService CartService) *RemoveProductsFromCartHandler {
	return &RemoveProductsFromCartHandler{cartService: cartService}
}

// @Summary      Удалить товар из корзины
// @Description  Метод полностью удаляет все количество товара из корзины пользователя.
// Если у пользователя вовсе нет данной позиции, то возвращается такой же ответ, как будто бы все позиции данного sku были успешно удалены
// @Tags         cart
// @Accept       json
// @Produce      json
// @Param        user_id  path  string  true  "Токен пользователя"
// @Param        sku_id   path  uint64  true  "SKU товара"
// @Success      200  {object}  RemoveProductsFromCartResponse
// @Failure      404  {object}  httpPkg.ErrorResponse
// @Router       /user/{user_id}/cart/{sku_id} [delete]
func (h *RemoveProductsFromCartHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	sku, err := parseSku(r)
	if err != nil {
		httpPkg.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	userId, err := parseUserId(r)
	if err != nil {
		httpPkg.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	err = h.cartService.RemoveProduct(r.Context(), userId, uint64(sku))
	if err != nil {
		httpPkg.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpPkg.WriteSuccessEmptyResponse(w)

	return
}

func parseSku(r *http.Request) (int, error) {
	skuRaw := r.PathValue("sku")
	sku, err := strconv.Atoi(skuRaw)
	if err != nil {
		return 0, errors.New("sku must be a number")
	}

	if sku < 1 {
		return 0, errors.New("sku must be more than zero")
	}

	return sku, nil
}

func parseUserId(r *http.Request) (uuid.UUID, error) {
	userIdRaw := r.PathValue("user_id")
	userId, err := uuid.Parse(userIdRaw)
	if err != nil {
		return uuid.Nil, errors.New("user_id must be valid uuid")
	}

	return userId, nil
}
