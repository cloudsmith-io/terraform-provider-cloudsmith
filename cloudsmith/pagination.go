package cloudsmith

import (
	"fmt"
	"net/http"
	"strconv"
)

const (
	DefaultPageSize int64 = 100

	paginationCountHeader     = "X-Pagination-Count"
	paginationPageHeader      = "X-Pagination-Page"
	paginationPageTotalHeader = "X-Pagination-PageTotal"
	paginationPageSizeHeader  = "X-Pagination-PageSize"
)

// PaginationOptions controls page size and optional result cap.
type PaginationOptions struct {
	PageSize   int64
	MaxResults int64
}

// PageFetcher fetches one page and returns the API-reported total page count.
type PageFetcher[T any] func(page, pageSize int64) (results []T, totalPages int64, err error)

// PageExecutor matches the generated SDK's list Execute methods.
type PageExecutor[T any] func(page, pageSize int64) (results []T, resp *http.Response, err error)

// PaginateAll collects every page reported by the API, stopping early only when
// MaxResults is reached. Iteration ends when the current page equals the
// API-reported total page count (X-Pagination-PageTotal).
func PaginateAll[T any](fetch PageFetcher[T], opts PaginationOptions) ([]T, error) {
	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}

	var all []T
	var page int64 = 1
	for {
		results, totalPages, err := fetch(page, pageSize)
		if err != nil {
			return nil, err
		}
		all = append(all, results...)
		if totalPages < 0 {
			return nil, fmt.Errorf("pagination returned invalid total page count %d", totalPages)
		}

		if opts.MaxResults > 0 && int64(len(all)) >= opts.MaxResults {
			return all[:opts.MaxResults], nil
		}

		if page >= totalPages {
			return all, nil
		}

		page++
	}
}

// PaginateAllHTTP validates Cloudsmith pagination headers before paginating.
// A 404 response is treated as an empty list so callers can short-circuit
// "not found" without hitting header-parsing errors.
func PaginateAllHTTP[T any](exec PageExecutor[T], opts PaginationOptions) ([]T, error) {
	fetch := func(page, pageSize int64) ([]T, int64, error) {
		results, resp, err := exec(page, pageSize)
		if err != nil {
			return nil, 0, err
		}
		if resp != nil && resp.StatusCode == http.StatusNotFound {
			return nil, 0, nil
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

// PaginateAllV2 walks a v2-SDK list-response chain by repeatedly calling
// next(currentResp). Iteration stops when next returns (nil, nil), which is
// the v2 SDK's convention for "no more pages" — the generated Next closure
// inspects the synthesised pagetotal field and short-circuits without an
// HTTP call when the chain is exhausted. extract pulls the typed result
// slice from each response. If maxResults > 0 and the running total exceeds
// it, the helper returns an error rather than silently truncating.
//
// Why callbacks rather than an interface: each v2 list operation returns a
// distinct concrete response type (e.g. WorkspacesPoliciesListResponse), and
// Go generics can't express the "self-returning Next" relationship. Two
// small lambdas at the call site are simpler than wrapping each response
// type in an adapter.
func PaginateAllV2[Resp any, T any](
	first *Resp,
	extract func(*Resp) []T,
	next func(*Resp) (*Resp, error),
	maxResults int64,
) ([]T, error) {
	var all []T
	cur := first
	for cur != nil {
		all = append(all, extract(cur)...)
		if maxResults > 0 && int64(len(all)) > maxResults {
			return nil, fmt.Errorf("result set exceeded the maximum supported size (%d)", maxResults)
		}
		nextResp, err := next(cur)
		if err != nil {
			return nil, err
		}
		cur = nextResp
	}
	return all, nil
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
