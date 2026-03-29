package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	errors2 "github.com/jva44ka/ozon-simulator-go-cart/internal/app/validation"
	domain_errors "github.com/jva44ka/ozon-simulator-go-cart/internal/model"
)

func WriteSuccessResponse(w http.ResponseWriter, response any) {
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(&response); err != nil {
		slog.Error("json.Encode failed", "err", err)
	}
}

func WriteSuccessEmptyResponse(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
}

func WriteErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	w.Header().Add("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(&ErrorResponse{Message: message}); err != nil {
		slog.Error("json.Encode failed", "err", err)
	}
}

func WriteServiceError(w http.ResponseWriter, err error) {
	var valErr *errors2.ValidationError
	switch {
	case errors.As(err, &valErr):
		WriteErrorResponse(w, http.StatusBadRequest, valErr.Error())
	case errors.Is(err, domain_errors.ErrProductNotFound), errors.Is(err, domain_errors.ErrCartItemsNotFound):
		WriteErrorResponse(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain_errors.ErrCartEmpty),
		errors.Is(err, domain_errors.ErrProductsCountMustBeGreaterThanNull):
		WriteErrorResponse(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain_errors.ErrInsufficientStock):
		WriteErrorResponse(w, http.StatusBadRequest, err.Error())
	default:
		WriteErrorResponse(w, http.StatusInternalServerError, err.Error())
	}
}
