package cloudsmith

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

// makeItems returns a slice of int identifiers [start, start+n).
func makeItems(start, n int64) []int64 {
	out := make([]int64, 0, n)
	for i := int64(0); i < n; i++ {
		out = append(out, start+i)
	}
	return out
}

func TestPaginateAll_MultiPageWithHeader(t *testing.T) {
	calls := 0
	fetch := func(page, pageSize int64) ([]int64, int64, error) {
		calls++
		switch page {
		case 1:
			return makeItems(0, pageSize), 3, nil
		case 2:
			return makeItems(pageSize, pageSize), 3, nil
		case 3:
			return makeItems(2*pageSize, pageSize), 3, nil
		}
		t.Fatalf("unexpected page %d", page)
		return nil, 0, nil
	}

	got, err := PaginateAll[int64](fetch, PaginationOptions{PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 30 {
		t.Fatalf("expected 30 results, got %d", len(got))
	}
	if calls != 3 {
		t.Fatalf("expected 3 fetcher calls, got %d", calls)
	}
}

func TestPaginateAll_SinglePageWithHeader(t *testing.T) {
	fetch := func(page, pageSize int64) ([]int64, int64, error) {
		if page != 1 {
			t.Fatalf("expected only page 1, got %d", page)
		}
		return makeItems(0, 3), 1, nil
	}

	got, err := PaginateAll[int64](fetch, PaginationOptions{PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 results, got %d", len(got))
	}
}

func TestPaginateAll_HitMaxResults(t *testing.T) {
	fetch := func(page, pageSize int64) ([]int64, int64, error) {
		return makeItems((page-1)*pageSize, pageSize), 100, nil
	}

	got, err := PaginateAll[int64](fetch, PaginationOptions{
		PageSize:   10,
		MaxResults: 25,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if int64(len(got)) != 25 {
		t.Fatalf("expected 25 results, got %d", len(got))
	}
}

func TestPaginateAll_MaxResultsStopsEarly(t *testing.T) {
	calls := 0
	fetch := func(page, pageSize int64) ([]int64, int64, error) {
		calls++
		return makeItems((page-1)*pageSize, pageSize), 9999, nil
	}

	got, err := PaginateAll[int64](fetch, PaginationOptions{
		PageSize:   10,
		MaxResults: 5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("expected 5 results, got %d", len(got))
	}
	if calls != 1 {
		t.Fatalf("expected 1 fetcher call, got %d", calls)
	}
}

func TestPaginateAll_FetcherErrorPropagates(t *testing.T) {
	sentinel := errors.New("api boom")
	fetch := func(page, pageSize int64) ([]int64, int64, error) {
		return nil, 0, sentinel
	}

	_, err := PaginateAll[int64](fetch, PaginationOptions{PageSize: 10})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}

func TestPaginateAll_DefaultsApplied(t *testing.T) {
	var seenPageSize int64
	fetch := func(page, pageSize int64) ([]int64, int64, error) {
		seenPageSize = pageSize
		return makeItems(0, 1), 1, nil
	}

	_, err := PaginateAll[int64](fetch, PaginationOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seenPageSize != DefaultPageSize {
		t.Fatalf("expected default pageSize %d, got %d", DefaultPageSize, seenPageSize)
	}
}

func TestPaginateAll_EmptyFirstPage(t *testing.T) {
	calls := 0
	fetch := func(page, pageSize int64) ([]int64, int64, error) {
		calls++
		return []int64{}, 0, nil
	}

	got, err := PaginateAll[int64](fetch, PaginationOptions{PageSize: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 results, got %d", len(got))
	}
	if calls != 1 {
		t.Fatalf("expected exactly 1 fetcher call, got %d", calls)
	}
}

func TestPaginateAll_MaxResultsZeroMeansUnlimited(t *testing.T) {
	fetch := func(page, pageSize int64) ([]int64, int64, error) {
		switch page {
		case 1:
			return makeItems(0, pageSize), 2, nil
		case 2:
			return makeItems(pageSize, pageSize), 2, nil
		}
		t.Fatalf("unexpected page %d", page)
		return nil, 0, nil
	}

	got, err := PaginateAll[int64](fetch, PaginationOptions{PageSize: 10, MaxResults: 0})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 20 {
		t.Fatalf("expected 20 results (no cap), got %d", len(got))
	}
}

func TestParsePageTotal(t *testing.T) {
	cases := []struct {
		name   string
		resp   *http.Response
		want   int64
		errSub string
	}{
		{name: "nil response", resp: nil, errSub: "missing HTTP response"},
		{name: "missing count header", resp: paginationResponse("7", withoutHeader(paginationCountHeader)), errSub: "missing required pagination header X-Pagination-Count"},
		{name: "missing page total header", resp: paginationResponse("7", withoutHeader(paginationPageTotalHeader)), errSub: "missing required pagination header X-Pagination-PageTotal"},
		{name: "valid headers", resp: paginationResponse("7"), want: 7},
		{name: "malformed page total header", resp: paginationResponse("not-a-number"), errSub: "parsing X-Pagination-PageTotal"},
		{name: "negative page total header", resp: paginationResponse("-1"), errSub: "value must be non-negative"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parsePageTotal(tc.resp)
			if tc.errSub != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.errSub)
				}
				if !strings.Contains(err.Error(), tc.errSub) {
					t.Fatalf("expected error containing %q, got %q", tc.errSub, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("want %d, got %d", tc.want, got)
			}
		})
	}
}

func paginationResponse(pageTotal string, opts ...func(http.Header)) *http.Response {
	header := http.Header{}
	header.Set(paginationCountHeader, "10")
	header.Set(paginationPageHeader, "1")
	header.Set(paginationPageTotalHeader, pageTotal)
	header.Set(paginationPageSizeHeader, "5")
	for _, opt := range opts {
		opt(header)
	}
	return &http.Response{Header: header}
}

func withoutHeader(name string) func(http.Header) {
	return func(header http.Header) {
		header.Del(name)
	}
}

func TestPaginateAllHTTP_ParsesHeader(t *testing.T) {
	calls := 0
	exec := func(page, pageSize int64) ([]int64, *http.Response, error) {
		calls++
		resp := paginationResponse("2")
		switch page {
		case 1:
			return makeItems(0, pageSize), resp, nil
		case 2:
			return makeItems(pageSize, pageSize), resp, nil
		}
		t.Fatalf("unexpected page %d", page)
		return nil, nil, nil
	}

	got, err := PaginateAllHTTP[int64](exec, PaginationOptions{PageSize: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 10 {
		t.Fatalf("expected 10 results, got %d", len(got))
	}
	if calls != 2 {
		t.Fatalf("expected 2 exec calls, got %d", calls)
	}
}

func TestPaginateAllHTTP_MissingHeaderErrors(t *testing.T) {
	calls := 0
	exec := func(page, pageSize int64) ([]int64, *http.Response, error) {
		calls++
		resp := &http.Response{Header: http.Header{}}
		return makeItems(0, pageSize), resp, nil
	}

	_, err := PaginateAllHTTP[int64](exec, PaginationOptions{PageSize: 5})
	if err == nil {
		t.Fatalf("expected missing header error, got nil")
	}
	if !strings.Contains(err.Error(), "missing required pagination header") {
		t.Fatalf("expected missing header error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 exec call, got %d", calls)
	}
}

func TestPaginateAllHTTP_NilResponseErrors(t *testing.T) {
	exec := func(page, pageSize int64) ([]int64, *http.Response, error) {
		return nil, nil, nil
	}

	_, err := PaginateAllHTTP[int64](exec, PaginationOptions{PageSize: 5})
	if err == nil {
		t.Fatalf("expected nil response error, got nil")
	}
	if !strings.Contains(err.Error(), "missing HTTP response") {
		t.Fatalf("expected nil response error, got %v", err)
	}
}

func TestPaginateAllHTTP_NotFoundReturnsEmpty(t *testing.T) {
	calls := 0
	exec := func(page, pageSize int64) ([]int64, *http.Response, error) {
		calls++
		resp := &http.Response{StatusCode: http.StatusNotFound, Header: http.Header{}}
		return nil, resp, nil
	}

	out, err := PaginateAllHTTP[int64](exec, PaginationOptions{PageSize: 5})
	if err != nil {
		t.Fatalf("expected nil error on 404, got %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty result on 404, got %d items", len(out))
	}
	if calls != 1 {
		t.Fatalf("expected 1 exec call, got %d", calls)
	}
}

func TestPaginateAllHTTP_ExecErrorPropagates(t *testing.T) {
	sentinel := errors.New("api boom")
	exec := func(page, pageSize int64) ([]int64, *http.Response, error) {
		return nil, nil, sentinel
	}

	_, err := PaginateAllHTTP[int64](exec, PaginationOptions{PageSize: 5})
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}

// v2Resp models a v2-SDK list response: a typed payload plus a Next closure
// that returns either the next response or (nil, nil) when exhausted.
type v2Resp struct {
	items []int64
	next  func() (*v2Resp, error)
}

func TestPaginateAllV2_WalksUntilNextReturnsNil(t *testing.T) {
	page3 := &v2Resp{items: []int64{4, 5}, next: func() (*v2Resp, error) { return nil, nil }}
	page2 := &v2Resp{items: []int64{2, 3}, next: func() (*v2Resp, error) { return page3, nil }}
	page1 := &v2Resp{items: []int64{0, 1}, next: func() (*v2Resp, error) { return page2, nil }}

	got, err := PaginateAllV2[v2Resp, int64](
		page1,
		func(r *v2Resp) []int64 { return r.items },
		func(r *v2Resp) (*v2Resp, error) { return r.next() },
		0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int64{0, 1, 2, 3, 4, 5}
	if len(got) != len(want) {
		t.Fatalf("expected %d results, got %d (%v)", len(want), len(got), got)
	}
	for i, v := range want {
		if got[i] != v {
			t.Fatalf("index %d: expected %d, got %d", i, v, got[i])
		}
	}
}

func TestPaginateAllV2_NilFirstResponse(t *testing.T) {
	got, err := PaginateAllV2[v2Resp, int64](
		nil,
		func(r *v2Resp) []int64 { return r.items },
		func(r *v2Resp) (*v2Resp, error) { return nil, nil },
		0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty result, got %v", got)
	}
}

func TestPaginateAllV2_NextErrorPropagates(t *testing.T) {
	sentinel := errors.New("next boom")
	page1 := &v2Resp{items: []int64{0, 1}, next: func() (*v2Resp, error) { return nil, sentinel }}

	_, err := PaginateAllV2[v2Resp, int64](
		page1,
		func(r *v2Resp) []int64 { return r.items },
		func(r *v2Resp) (*v2Resp, error) { return r.next() },
		0,
	)
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected sentinel error, got %v", err)
	}
}

func TestPaginateAllV2_MaxResultsExceededErrors(t *testing.T) {
	page2 := &v2Resp{items: []int64{2, 3}, next: func() (*v2Resp, error) { return nil, nil }}
	page1 := &v2Resp{items: []int64{0, 1}, next: func() (*v2Resp, error) { return page2, nil }}

	_, err := PaginateAllV2[v2Resp, int64](
		page1,
		func(r *v2Resp) []int64 { return r.items },
		func(r *v2Resp) (*v2Resp, error) { return r.next() },
		3,
	)
	if err == nil {
		t.Fatal("expected overflow error, got nil")
	}
	if !strings.Contains(err.Error(), "maximum supported size") {
		t.Fatalf("expected overflow error mentioning maximum size, got %v", err)
	}
}

func TestPaginateAllV2_MaxResultsAtBoundaryReturnsAll(t *testing.T) {
	// Exactly hitting maxResults (not exceeding) is allowed; the helper
	// only errors when the running total > maxResults.
	page2 := &v2Resp{items: []int64{2, 3}, next: func() (*v2Resp, error) { return nil, nil }}
	page1 := &v2Resp{items: []int64{0, 1}, next: func() (*v2Resp, error) { return page2, nil }}

	got, err := PaginateAllV2[v2Resp, int64](
		page1,
		func(r *v2Resp) []int64 { return r.items },
		func(r *v2Resp) (*v2Resp, error) { return r.next() },
		4,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("expected 4 results, got %d", len(got))
	}
}
