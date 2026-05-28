package handler

import (
	"net/url"
	"strconv"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// ParsePagination extracts page and page_size from query params.
// Accepts "page_size" (canonical) with "pageSize" and "per_page" as fallbacks
// for backward compatibility.
func ParsePagination(query url.Values) (page, pageSize int) {
	page, _ = strconv.Atoi(query.Get("page"))

	pageSize, _ = strconv.Atoi(query.Get("page_size"))
	if pageSize == 0 {
		pageSize, _ = strconv.Atoi(query.Get("pageSize"))
	}
	if pageSize == 0 {
		pageSize, _ = strconv.Atoi(query.Get("per_page"))
	}

	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 || pageSize > maxPageSize {
		pageSize = defaultPageSize
	}
	return
}
