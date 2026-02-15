package clean_cart_handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	httpPkg "github.com/jva44ka/ozon-simulator-go-cart/pkg/http"
)

type CartService interface {
	RemoveAllProducts(ctx context.Context, userId uuid.UUID) error
}

type CleanCartHandler struct {
	cartService CartService
}

func NewCleanCartHandler(cartService CartService) *CleanCartHandler {
	return &CleanCartHandler{cartService: cartService}
}

// @Summary      Очистить корзину пользователя
// @Description  Метод полностью очищает корзину пользователя.
// Если у пользователя нет корзины или она пуста, то, как и при успешной очистке корзины, необходимо вернуть код ответа 204 No Content.
// @Tags         cart
// @Accept       json
// @Produce      json
// @Param        user_id  path  string  true  "Токен пользователя"
// @Success      200  {object}  CleanCartResponse
// @Failure      404  {object}  httpPkg.ErrorResponse
// @Router       /user/{user_id}/cart [delete]
func (h *CleanCartHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userId, err := parseUserId(r)
	if err != nil {
		httpPkg.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	err = h.cartService.RemoveAllProducts(r.Context(), userId)
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
