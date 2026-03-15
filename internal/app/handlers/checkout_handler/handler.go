package checkout_handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	httpPkg "github.com/jva44ka/ozon-simulator-go-cart/pkg/http"
)

type CartService interface {
	Checkout(ctx context.Context, userId uuid.UUID) error
}

type CheckoutHandler struct {
	cartService CartService
}

func NewCheckoutHandler(cartService CartService) *CheckoutHandler {
	return &CheckoutHandler{cartService: cartService}
}

// @Summary      Оформить заказ
// @Description  Метод оформляет заказ пользователя: списывает товары со склада и очищает корзину.
// Если корзина пользователя пуста, возвращается ошибка.
// @Tags         cart
// @Accept       json
// @Produce      json
// @Param        user_id  path  string  true  "Токен пользователя"
// @Success      200  {object}  CheckoutResponse
// @Failure      400  {object}  httpPkg.ErrorResponse
// @Failure      404  {object}  httpPkg.ErrorResponse
// @Failure      500  {object}  httpPkg.ErrorResponse
// @Router       /user/{user_id}/cart/checkout [post]
func (h *CheckoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userId, err := parseUserId(r)
	if err != nil {
		httpPkg.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	err = h.cartService.Checkout(r.Context(), userId)
	if err != nil {
		httpPkg.WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	httpPkg.WriteSuccessEmptyResponse(w)
}

func parseUserId(r *http.Request) (uuid.UUID, error) {
	userIdRaw := r.PathValue("user_id")
	userId, err := uuid.Parse(userIdRaw)
	if err != nil {
		return uuid.Nil, errors.New("user_id must be valid uuid")
	}

	return userId, nil
}
