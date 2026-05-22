package cloudsmith

import (
	"fmt"
	"net/http"
	"strconv"
)

const (
	DefaultPageSize int64 = 100
	DefaultMaxPages int64 = 650

	paginationCountHeader     = "X-Pagination-Count"
	paginationPageHeader      = "X-Pagination-Page"
	paginationPageTotalHeader = "X-Pagination-PageTotal"
	paginationPageSizeHeader  = "X-Pagination-PageSize"
)

// PaginationOptions controls page size and result caps.
type PaginationOptions struct {
	PageSize   int64
	MaxPages   int64
	MaxResults int64
}

// PageFetcher fetches one page and returns the API-reported total page count.
type PageFetcher[T any] func(page, pageSize int64) (results []T, totalPages int64, err error)

// PageExecutor matches the generated SDK's list Execute methods.
type PageExecutor[T any] func(page, pageSize int64) (results []T, resp *http.Response, err error)

// PaginateAll collects pages until MaxResults or the API-reported page total.
func PaginateAll[T any](fetch PageFetcher[T], opts PaginationOptions) ([]T, error) {
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	maxPages := opts.MaxPages
	if maxPages <= 0 {
		maxPages = DefaultMaxPages
	}

	var all []T
	var totalPages int64
	var page int64 = 1
	for {
		if page > maxPages {
			return nil, fmt.Errorf(
				"pagination exceeded MaxPages (%d); aborting to prevent runaway iteration",
				maxPages,
			)
		}

		results, tp, err := fetch(page, pageSize)
		if err != nil {
			return nil, err
		}
		all = append(all, results...)
		if tp < 0 {
			return nil, fmt.Errorf("pagination returned invalid total page count %d", tp)
		}
		totalPages = tp

		if opts.MaxResults > 0 && int64(len(all)) >= opts.MaxResults {
			return all[:opts.MaxResults], nil
		}

		if totalPages > maxPages && (opts.MaxResults <= 0 || maxPages*pageSize < opts.MaxResults) {
			return nil, fmt.Errorf(
				"pagination requires %d pages, exceeding MaxPages (%d)",
				totalPages,
				maxPages,
			)
		}

		if page >= totalPages {
			return all, nil
		}

		page++
	}
}

// PaginateAllHTTP validates Cloudsmith pagination headers before paginating.
func PaginateAllHTTP[T any](exec PageExecutor[T], opts PaginationOptions) ([]T, error) {
	fetch := func(page, pageSize int64) ([]T, int64, error) {
		results, resp, err := exec(page, pageSize)
		if err != nil {
			return nil, 0, err
		}
		total, err := parsePageTotal(resp)
		if err != nil {
			return nil, 0, err
		}
		return results, total, nil
	}
	return PaginateAll[T](fetch, opts)
}

// parsePageTotal validates the Cloudsmith pagination headers and returns PageTotal.
func parsePageTotal(resp *http.Response) (int64, error) {
	if resp == nil {
		return 0, fmt.Errorf("missing HTTP response while parsing pagination headers")
	}
	if _, err := parseRequiredPaginationHeader(resp, paginationCountHeader); err != nil {
		return 0, err
	}
	if _, err := parseRequiredPaginationHeader(resp, paginationPageHeader); err != nil {
		return 0, err
	}
	total, err := parseRequiredPaginationHeader(resp, paginationPageTotalHeader)
	if err != nil {
		return 0, err
	}
	if _, err := parseRequiredPaginationHeader(resp, paginationPageSizeHeader); err != nil {
		return 0, err
	}
	return total, nil
}

func parseRequiredPaginationHeader(resp *http.Response, name string) (int64, error) {
	raw := resp.Header.Get(name)
	if raw == "" {
		return 0, fmt.Errorf("missing required pagination header %s", name)
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing %s: %w", name, err)
	}
	if n < 0 {
		return 0, fmt.Errorf("parsing %s: value must be non-negative", name)
	}
	return n, nil
}
