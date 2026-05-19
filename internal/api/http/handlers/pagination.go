package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/sevoniva/nivora/internal/api/http/dto"
)

const (
	MaxRequestBodyBytes = 4 << 20
	MaxLogChunkBytes    = 64 << 10
	DefaultPageLimit    = 100
	MaxPageLimit        = 500
)

type paginationOptions struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	Total   int  `json:"total"`
	Enabled bool `json:"-"`
}

func parsePagination(r *http.Request) (paginationOptions, error) {
	query := r.URL.Query()
	limitText := query.Get("limit")
	offsetText := query.Get("offset")
	if limitText == "" && offsetText == "" {
		return paginationOptions{}, nil
	}
	limit := DefaultPageLimit
	if limitText != "" {
		parsed, err := strconv.Atoi(limitText)
		if err != nil || parsed <= 0 {
			return paginationOptions{}, fmt.Errorf("limit must be a positive integer")
		}
		if parsed > MaxPageLimit {
			return paginationOptions{}, fmt.Errorf("limit must be <= %d", MaxPageLimit)
		}
		limit = parsed
	}
	offset := 0
	if offsetText != "" {
		parsed, err := strconv.Atoi(offsetText)
		if err != nil || parsed < 0 {
			return paginationOptions{}, fmt.Errorf("offset must be a non-negative integer")
		}
		offset = parsed
	}
	return paginationOptions{Limit: limit, Offset: offset, Enabled: true}, nil
}

func paginatedPayload[T any](items []T, page paginationOptions) any {
	if !page.Enabled {
		return items
	}
	page.Total = len(items)
	start := page.Offset
	if start > len(items) {
		start = len(items)
	}
	end := start + page.Limit
	if end > len(items) {
		end = len(items)
	}
	return map[string]any{
		"items":      items[start:end],
		"pagination": page,
	}
}

func respondPaginated[T any](w http.ResponseWriter, r *http.Request, items []T, err error) bool {
	if err != nil {
		return false
	}
	page, pageErr := parsePagination(r)
	if pageErr != nil {
		RespondError(w, r, http.StatusBadRequest, dto.ErrorResponse{Code: "invalid_pagination", Message: pageErr.Error()})
		return true
	}
	RespondJSON(w, http.StatusOK, paginatedPayload(items, page))
	return true
}
