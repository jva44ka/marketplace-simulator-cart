package checkout_handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	httpPkg "github.com/jva44ka/marketplace-simulator-cart/pkg/http"
)

type CheckoutUseCase interface {
	Execute(ctx context.Context, userId uuid.UUID) (float64, error)
}

type Validator interface {
	GetValidatedSku(skuRaw string) (uint64, error)
	GetValidatedUserId(userIdRaw string) (uuid.UUID, error)
}

type CheckoutHandler struct {
	useCase   CheckoutUseCase
	validator Validator
}

func NewCheckoutHandler(useCase CheckoutUseCase, validator Validator) *CheckoutHandler {
	return &CheckoutHandler{
		useCase:   useCase,
		validator: validator,
	}
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

	userIdRaw := r.PathValue("user_id")
	userId, err := h.validator.GetValidatedUserId(userIdRaw)
	if err != nil {
		httpPkg.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	totalPrice, err := h.useCase.Execute(r.Context(), userId)
	if err != nil {
		httpPkg.WriteServiceError(w, err)
		return
	}

	response := CheckoutResponse{
		TotalPrice: totalPrice,
	}

	httpPkg.WriteSuccessResponse(w, response)
}
