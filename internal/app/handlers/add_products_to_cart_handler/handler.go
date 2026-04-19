package add_products_to_cart_handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	httpPkg "github.com/jva44ka/marketplace-simulator-cart/pkg/http"
)

type CartService interface {
	AddProduct(ctx context.Context, userId uuid.UUID, sku uint64, count uint32) error
}

type Validator interface {
	GetValidatedSku(skuRaw string) (uint64, error)
	GetValidatedUserId(userIdRaw string) (uuid.UUID, error)
}

type AddProductsToCartHandler struct {
	cartService CartService
	validator   Validator
}

func NewAddProductsToCartHandler(cartService CartService, validator Validator) *AddProductsToCartHandler {
	return &AddProductsToCartHandler{
		cartService: cartService,
		validator:   validator,
	}
}

// @Summary      Добавить товар в корзину
// @Description  Идентификатором товара является числовой идентификатор SKU.
// Метод добавляет указанный товар в корзину определенного пользователя.
// Каждый пользователь имеет числовой идентификатор userID.
// При добавлении в корзину проверяем, что товар существует в специальном сервисе.
// Один и тот же товар может быть добавлен в корзину несколько раз, при этом количество экземпляров складывается.
// @Tags         cart
// @Accept       json
// @Produce      json
// @Param        user_id  path  string  true  "Токен пользователя"
// @Param        sku   path  uint64  true  "SKU товара"
// @Param        body     body  AddProductToCartRequest  true  "Тело запроса с количеством товаров"
// @Success      200  {object}  AddProductToCartResponse
// @Failure      404  {object}  httpPkg.ErrorResponse
// @Router       /user/{user_id}/cart/{sku} [post]
func (h *AddProductsToCartHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	var request AddProductToCartRequest

	if err = json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpPkg.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	err = h.cartService.AddProduct(r.Context(), userId, uint64(sku), request.Count)
	if err != nil {
		httpPkg.WriteServiceError(w, err)
		return
	}

	httpPkg.WriteSuccessEmptyResponse(w)
	return
}
