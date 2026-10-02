package helpers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/ortizdavid/go-bank-core-api/common/apperrors"
)

const (
	defaultPageLimit = 10
	maxPageLimit     = 100
)

type PaginationParams struct {
	CurrentPage int
	Limit       int
}

// GetPaginationParams extracts current page and limit from URL query parameters.
func GetPaginationParams(r *http.Request) (PaginationParams, error) {
	currentPage, err := queryInt(r, "current_page", 0)
	if err != nil {
		return PaginationParams{}, err
	}
	limit, err := queryInt(r, "limit", defaultPageLimit)
	if err != nil {
		return PaginationParams{}, err
	}
	if limit > maxPageLimit {
		return PaginationParams{}, apperrors.NewBadRequestError(fmt.Sprintf("limit must be at most %d", maxPageLimit))
	}
	return PaginationParams{
		CurrentPage: currentPage,
		Limit:       limit,
	}, nil
}

// PathInt64 reads a numeric path parameter.
func PathInt64(r *http.Request, name string) (int64, error) {
	value, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil {
		return 0, apperrors.NewBadRequestError(name + " must be an integer")
	}
	return value, nil
}

func queryInt(r *http.Request, name string, fallback int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, apperrors.NewBadRequestError(name + " must be an integer")
	}
	return value, nil
}
