package helpers

import (
	"errors"
	"net/http"

	"github.com/ortizdavid/go-bank-core-api/common/apperrors"
	"github.com/ortizdavid/go-nopain/httputils"
)

// HandleHttpErrors centralizes error handling for http-related operations.
// Validation and domain errors keep their status codes. Unexpected errors are
// returned as a generic 500 so database details are never sent to clients.
func HandleHttpErrors(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	var httpErr *apperrors.HttpError
	if errors.As(err, &httpErr) {
		httputils.WriteJsonError(w, httpErr.Error(), httpErr.StatusCode)
		return
	}
	httputils.WriteJsonError(w, "Internal Server Error: an unexpected error occurred", http.StatusInternalServerError)
}
