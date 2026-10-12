package network_api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// pageServer serves one canned JSON page per request, in order, and records
// every request URL. Requests past the last page fail the test.
type pageServer struct {
	t     *testing.T
	pages []string

	mu       sync.Mutex
	requests []*url.URL
}

func newPageServer(t *testing.T, pages ...string) (*pageServer, *ClientWithResponses) {
	t.Helper()

	ps := &pageServer{t: t, pages: pages}
	srv := httptest.NewServer(ps)
	t.Cleanup(srv.Close)

	api, err := NewAPI("key", srv.URL)
	if err != nil {
		t.Fatalf("NewAPI: %v", err)
	}

	return ps, api
}

func (ps *pageServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ps.mu.Lock()
	ps.requests = append(ps.requests, r.URL)
	n := len(ps.requests)
	ps.mu.Unlock()

	if n > len(ps.pages) {
		ps.t.Errorf("request %d (%s) past the last page", n, r.URL)
		w.WriteHeader(http.StatusTeapot)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(ps.pages[n-1]))
}

func (ps *pageServer) seen() []*url.URL {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	return append([]*url.URL(nil), ps.requests...)
}

func fabricsPage(t *testing.T, next string, slugs ...string) string {
	t.Helper()

	items := make([]FabricResponse, 0, len(slugs))
	for _, s := range slugs {
		items = append(items, FabricResponse{Slug: s, DisplayName: strings.ToUpper(s), Metro: "metro-" + s})
	}

	links := map[string]any{"self": map[string]string{"href": "/fabrics?limit=100"}}
	if next != "" {
		links["next"] = map[string]string{"href": next}
	}

	b, err := json.Marshal(map[string]any{"items": items, "links": links})
	if err != nil {
		t.Fatalf("marshal page: %v", err)
	}

	return string(b)
}

func TestListAllFabricsWalksRelativeNextLinks(t *testing.T) {
	t.Parallel()

	ps, api := newPageServer(t,
		fabricsPage(t, "/fabrics?limit=100&cursor=abc", "rotterdam-a", "rotterdam-b"),
		fabricsPage(t, "/fabrics?limit=100&cursor=def", "frankfurt-a"),
		fabricsPage(t, "", "tokyo-a"),
	)

	fabrics, err := api.ListAllFabrics(context.Background())
	if err != nil {
		t.Fatalf("ListAllFabrics: %v", err)
	}

	var slugs []string
	for _, f := range fabrics {
		slugs = append(slugs, f.Slug)
	}
	if got, want := strings.Join(slugs, ","), "rotterdam-a,rotterdam-b,frankfurt-a,tokyo-a"; got != want {
		t.Errorf("slugs = %s, want %s (all pages, in order)", got, want)
	}
	if fabrics[0].DisplayName != "ROTTERDAM-A" || fabrics[0].Metro != "metro-rotterdam-a" {
		t.Errorf("first fabric = %+v, want display name and metro decoded", fabrics[0])
	}

	reqs := ps.seen()
	if len(reqs) != 3 {
		t.Fatalf("made %d requests, want 3", len(reqs))
	}

	wantCursors := []string{"", "abc", "def"}
	for i, u := range reqs {
		if u.Path != "/network/v1/fabrics" {
			t.Errorf("request %d path = %q, want /network/v1/fabrics (never the bare href)", i+1, u.Path)
		}
		if got := u.Query().Get("cursor"); got != wantCursors[i] {
			t.Errorf("request %d cursor = %q, want %q", i+1, got, wantCursors[i])
		}
		if got := u.Query().Get("limit"); got != "100" {
			t.Errorf("request %d limit = %q, want 100", i+1, got)
		}
	}
}

// The cursor carries no filters, so every page request must repeat them.
func TestWalkPagesRepeatsFiltersAndLimitOnEveryPage(t *testing.T) {
	t.Parallel()

	page := func(next string) string {
		links := `"self":{"href":"/networks?limit=100"}`
		if next != "" {
			links += `,"next":{"href":"` + next + `"}`
		}
		return `{"items":[],"links":{` + links + `}}`
	}

	ps, api := newPageServer(t,
		page("/networks?limit=100&cursor=o%3D100"),
		page("/networks?limit=100&cursor=o%3D200"),
		page(""),
	)

	fabric := "rotterdam-a"
	_, err := WalkPages(context.Background(), func(ctx context.Context, cursor *string) ([]NetworkResponse, *Link, error) {
		limit := PageSize
		resp, err := api.ListNetworksWithResponse(ctx, &ListNetworksParams{
			Limit:           &limit,
			Cursor:          cursor,
			FiltersFabricEq: &fabric,
		})
		if err != nil {
			return nil, nil, err
		}
		if err := CheckResponse(resp.StatusCode(), resp.Body); err != nil {
			return nil, nil, err
		}
		return resp.JSON200.Items, resp.JSON200.Links.Next, nil
	})
	if err != nil {
		t.Fatalf("WalkPages: %v", err)
	}

	reqs := ps.seen()
	if len(reqs) != 3 {
		t.Fatalf("made %d requests, want 3", len(reqs))
	}

	wantCursors := []string{"", "o=100", "o=200"}
	for i, u := range reqs {
		q := u.Query()
		if u.Path != "/network/v1/networks" {
			t.Errorf("request %d path = %q, want /network/v1/networks", i+1, u.Path)
		}
		if got := q.Get("filters[fabric][eq]"); got != fabric {
			t.Errorf("request %d filters[fabric][eq] = %q, want %q", i+1, got, fabric)
		}
		if got := q.Get("limit"); got != "100" {
			t.Errorf("request %d limit = %q, want 100", i+1, got)
		}
		if got := q.Get("cursor"); got != wantCursors[i] {
			t.Errorf("request %d cursor = %q, want %q", i+1, got, wantCursors[i])
		}
	}
}

func TestListAllFabricsReturnsAnEmptySliceForAnEmptyListing(t *testing.T) {
	t.Parallel()

	_, api := newPageServer(t, `{"items":[],"links":{"self":{"href":"/fabrics?limit=100"}}}`)

	fabrics, err := api.ListAllFabrics(context.Background())
	if err != nil {
		t.Fatalf("ListAllFabrics: %v", err)
	}
	if fabrics == nil || len(fabrics) != 0 {
		t.Errorf("fabrics = %#v, want an empty, non-nil slice", fabrics)
	}
}

func TestListAllFabricsReturnsTheAPIErrorOfAnyPage(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"slug":"a","displayName":"A","metro":"M"}],"links":{"self":{"href":"/fabrics"},"next":{"href":"/fabrics?cursor=abc"}}}`))
			return
		}
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"title":"Internal Server Error","status":500,"code":"E000001","traceId":"trace-2"}`))
	}))
	t.Cleanup(srv.Close)

	api, err := NewAPI("key", srv.URL)
	if err != nil {
		t.Fatalf("NewAPI: %v", err)
	}

	fabrics, err := api.ListAllFabrics(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("ListAllFabrics: %v, want the page-2 *APIError", err)
	}
	if apiErr.TraceID != "trace-2" {
		t.Errorf("TraceID = %q, want trace-2", apiErr.TraceID)
	}
	if fabrics != nil {
		t.Errorf("fabrics = %v, want nil on error (no partial listing)", fabrics)
	}
}

// fakeFetch serves canned pages to WalkPages and counts the calls, so a
// broken next link shows up as a bounded failure, not a hang.
func fakeFetch(pages []*Link, calls *int) FetchPage[int] {
	return func(_ context.Context, _ *string) ([]int, *Link, error) {
		*calls++
		if *calls > len(pages) {
			return nil, nil, errors.New("walked past the canned pages")
		}
		return []int{*calls}, pages[*calls-1], nil
	}
}

func TestWalkPagesRefusesBrokenNextLinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		pages     []*Link
		wantCalls int
		wantErr   string
	}{
		{
			name:      "next link without a cursor",
			pages:     []*Link{{Href: "/fabrics?limit=100"}},
			wantCalls: 1,
			wantErr:   "carries no cursor",
		},
		{
			name:      "next link with an empty cursor",
			pages:     []*Link{{Href: "/fabrics?limit=100&cursor="}},
			wantCalls: 1,
			wantErr:   "carries no cursor",
		},
		{
			name: "cursor repeated",
			pages: []*Link{
				{Href: "/fabrics?cursor=abc"},
				{Href: "/fabrics?cursor=def"},
				{Href: "/fabrics?cursor=abc"},
			},
			wantCalls: 3,
			wantErr:   "twice",
		},
		{
			name:      "unparseable href",
			pages:     []*Link{{Href: "%zz"}},
			wantCalls: 1,
			wantErr:   "could not parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var calls int
			items, err := WalkPages(context.Background(), fakeFetch(tt.pages, &calls))
			if err == nil {
				t.Fatalf("WalkPages returned %v, want an error", items)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("err = %v, want it to mention %q", err, tt.wantErr)
			}
			if calls != tt.wantCalls {
				t.Errorf("fetched %d pages, want %d", calls, tt.wantCalls)
			}
		})
	}
}

func TestWalkPagesPassesNilCursorFirstThenTheParsedCursor(t *testing.T) {
	t.Parallel()

	var got []string
	pages := []*Link{{Href: "/x?cursor=first%2Bone"}, nil}
	calls := 0

	items, err := WalkPages(context.Background(), func(_ context.Context, cursor *string) ([]string, *Link, error) {
		if cursor == nil {
			got = append(got, "<nil>")
		} else {
			got = append(got, *cursor)
		}
		calls++
		return []string{"item"}, pages[calls-1], nil
	})
	if err != nil {
		t.Fatalf("WalkPages: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("items = %v, want one per page", items)
	}
	if strings.Join(got, ",") != "<nil>,first+one" {
		t.Errorf("cursors = %v, want <nil> then the decoded cursor first+one", got)
	}
}

// The status decides before the body: a gateway error with a JSON content
// type and a body that is not JSON is an *APIError, not a decode error.
func TestListAllFabricsReportsANon2xxWithABrokenJSONBodyAsAnAPIError(t *testing.T) {
	t.Parallel()

	for _, body := range []string{`{"items":[`, ""} {
		t.Run(fmt.Sprintf("body %q", body), func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadGateway)
				_, _ = w.Write([]byte(body))
			}))
			t.Cleanup(srv.Close)

			api, err := NewAPI("key", srv.URL)
			if err != nil {
				t.Fatalf("NewAPI: %v", err)
			}

			_, err = api.ListAllFabrics(context.Background())
			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("ListAllFabrics: %v, want an *APIError", err)
			}
			if apiErr.StatusCode != http.StatusBadGateway {
				t.Errorf("StatusCode = %d, want 502", apiErr.StatusCode)
			}
			if errors.Is(err, ErrNoResponse) {
				t.Errorf("err = %v, a 502 is a response, not ErrNoResponse", err)
			}
		})
	}
}

func TestListAllFabricsReportsA200ThatDoesNotDecodeAsErrNoResponse(t *testing.T) {
	t.Parallel()

	_, api := newPageServer(t, `{"items":[{"slug":`)

	fabrics, err := api.ListAllFabrics(context.Background())
	if !errors.Is(err, ErrNoResponse) {
		t.Fatalf("ListAllFabrics: %v, want it to wrap ErrNoResponse", err)
	}
	if fabrics != nil {
		t.Errorf("fabrics = %v, want nil on error", fabrics)
	}
}
