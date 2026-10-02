package apperrors

import (
	"net/http"
)

func NewInternalError(message string) *HttpError {
	return &HttpError{
		Message:    message,
		StatusCode: http.StatusInternalServerError,
	}
}
