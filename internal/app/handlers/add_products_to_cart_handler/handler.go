package add_products_to_cart_handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	httpPkg "github.com/jva44ka/ozon-simulator-go-cart/pkg/http"
)

type CartService interface {
	AddProduct(ctx context.Context, userId uuid.UUID, sku uint64, count uint32) error
}

type AddProductsToCartHandler struct {
	cartService CartService
}

func NewAddProductsToCartHandler(cartService CartService) *AddProductsToCartHandler {
	return &AddProductsToCartHandler{cartService: cartService}
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
// @Param        sku_id   path  uint64  true  "SKU товара"
// @Param        body     body  AddProductToCartRequest  true  "Тело запроса с количеством товаров"
// @Success      200  {object}  AddProductToCartResponse
// @Failure      404  {object}  httpPkg.ErrorResponse
// @Router       /user/{user_id}/cart/{sku_id} [post]
func (h *AddProductsToCartHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	var request AddProductToCartRequest

	if err = json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpPkg.WriteErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	err = h.cartService.AddProduct(r.Context(), userId, uint64(sku), request.Count)
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
