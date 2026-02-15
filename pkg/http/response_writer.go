package http

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func WriteSuccessResponse(w http.ResponseWriter, response any) {
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(&response); err != nil {
		fmt.Println("json.Encode failed")
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
		fmt.Println("json.Encode failed")
	}
}
