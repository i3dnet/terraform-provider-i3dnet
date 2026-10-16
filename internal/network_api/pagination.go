package network_api

import (
	"context"
	"errors"
	"fmt"
	"net/url"
)

// PageSize is the limit every listing requests.
const PageSize = 100

// FetchPage requests one page of a listing. cursor is nil for the first page.
// The closure owns the filters and the limit, so every page of a walk is
// requested with the same ones. It returns the page's items and its
// links.next, nil on the last page.
type FetchPage[T any] func(ctx context.Context, cursor *string) (items []T, next *Link, err error)

// WalkPages requests every page of a listing and returns the items in order,
// an empty slice when there are none.
//
// The next page is requested through fetch with the cursor query parameter of
// links.next.href, never by requesting the href itself: hrefs are relative to
// the API base and omit /network/v1, and the cursor carries no filters.
func WalkPages[T any](ctx context.Context, fetch FetchPage[T]) ([]T, error) {
	all := []T{}
	var cursor *string
	seen := map[string]bool{}

	for {
		items, next, err := fetch(ctx, cursor)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)

		if next == nil {
			return all, nil
		}

		nextCursor, err := cursorFromHref(next.Href)
		if err != nil {
			return nil, err
		}
		if seen[nextCursor] {
			return nil, fmt.Errorf("listing returned cursor %q twice", nextCursor)
		}
		seen[nextCursor] = true
		cursor = &nextCursor
	}
}

func cursorFromHref(href string) (string, error) {
	u, err := url.Parse(href)
	if err != nil {
		return "", fmt.Errorf("could not parse links.next.href %q: %w", href, err)
	}

	cursor := u.Query().Get("cursor")
	if cursor == "" {
		return "", errors.New("links.next.href " + href + " carries no cursor")
	}

	return cursor, nil
}

// ListAllFabrics returns every public fabric.
func (c *ClientWithResponses) ListAllFabrics(ctx context.Context) ([]FabricResponse, error) {
	return WalkPages(ctx, func(ctx context.Context, cursor *string) ([]FabricResponse, *Link, error) {
		limit := PageSize
		page, err := Do[FabricListResponse](c.ListFabrics(ctx, &ListFabricsParams{Limit: &limit, Cursor: cursor}))
		if err != nil {
			return nil, nil, err
		}

		return page.Items, page.Links.Next, nil
	})
}

// ListAllNetworks returns every network of the account that matches the
// filters set in params. Limit and Cursor in params are ignored: every page is
// requested with PageSize and the same filters.
func (c *ClientWithResponses) ListAllNetworks(ctx context.Context, params ListNetworksParams) ([]NetworkResponse, error) {
	return WalkPages(ctx, func(ctx context.Context, cursor *string) ([]NetworkResponse, *Link, error) {
		limit := PageSize
		pageParams := params
		pageParams.Limit = &limit
		pageParams.Cursor = cursor

		page, err := Do[NetworkListResponse](c.ListNetworks(ctx, &pageParams))
		if err != nil {
			return nil, nil, err
		}

		return page.Items, page.Links.Next, nil
	})
}

// ListAllPrefixes returns every prefix of the account that matches the
// filters set in params. Limit and Cursor in params are ignored: every page is
// requested with PageSize and the same filters.
func (c *ClientWithResponses) ListAllPrefixes(ctx context.Context, params ListPrefixesParams) ([]PrefixResponse, error) {
	return WalkPages(ctx, func(ctx context.Context, cursor *string) ([]PrefixResponse, *Link, error) {
		limit := PageSize
		pageParams := params
		pageParams.Limit = &limit
		pageParams.Cursor = cursor

		page, err := Do[PrefixListResponse](c.ListPrefixes(ctx, &pageParams))
		if err != nil {
			return nil, nil, err
		}

		return page.Items, page.Links.Next, nil
	})
}
