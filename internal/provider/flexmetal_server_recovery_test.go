package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"terraform-provider-i3dnet/internal/one_api"
)

func testServer(uuid, name, status string, createdAt int64) one_api.Server {
	s := one_api.Server{
		Uuid:      uuid,
		Name:      name,
		Status:    status,
		CreatedAt: createdAt,
	}
	s.Location.Name = "EU: Rotterdam"
	s.InstanceType.Name = "bm7.std.8"
	return s
}

func TestSelectCreatedServer(t *testing.T) {
	criteria := createdServerCriteria{
		Name:         "wk-0054",
		Location:     "EU: Rotterdam",
		InstanceType: "bm7.std.8",
		NotBefore:    1000,
	}

	releasedServer := testServer("released", "wk-0054", "released", 1100)
	releasedServer.ReleasedAt = 1150

	otherLocation := testServer("other-location", "wk-0054", "created", 1100)
	otherLocation.Location.Name = "AU: Sydney"

	otherInstanceType := testServer("other-instance-type", "wk-0054", "created", 1100)
	otherInstanceType.InstanceType.Name = "bm9.hmm.12"

	tests := map[string]struct {
		servers        []one_api.Server
		wantMatch      string
		wantCandidates int
	}{
		"single match is adopted": {
			servers:        []one_api.Server{testServer("a", "wk-0054", "created", 1100)},
			wantMatch:      "a",
			wantCandidates: 1,
		},
		"match already progressed past created is adopted": {
			servers:        []one_api.Server{testServer("a", "wk-0054", "provisioning", 1100)},
			wantMatch:      "a",
			wantCandidates: 1,
		},
		"no servers at all": {
			servers:        nil,
			wantMatch:      "",
			wantCandidates: 0,
		},
		"different name is ignored": {
			servers:        []one_api.Server{testServer("a", "wk-0055", "created", 1100)},
			wantMatch:      "",
			wantCandidates: 0,
		},
		"server created before the window is ignored": {
			servers:        []one_api.Server{testServer("a", "wk-0054", "created", 999)},
			wantMatch:      "",
			wantCandidates: 0,
		},
		"server created exactly at the window start is a match": {
			servers:        []one_api.Server{testServer("a", "wk-0054", "created", 1000)},
			wantMatch:      "a",
			wantCandidates: 1,
		},
		"released server is ignored": {
			servers:        []one_api.Server{releasedServer},
			wantMatch:      "",
			wantCandidates: 0,
		},
		"releasing server is ignored": {
			servers:        []one_api.Server{testServer("a", "wk-0054", "releasing", 1100)},
			wantMatch:      "",
			wantCandidates: 0,
		},
		"different location is ignored": {
			servers:        []one_api.Server{otherLocation},
			wantMatch:      "",
			wantCandidates: 0,
		},
		"different instance type is ignored": {
			servers:        []one_api.Server{otherInstanceType},
			wantMatch:      "",
			wantCandidates: 0,
		},
		"duplicate names are disambiguated by the created status": {
			servers: []one_api.Server{
				testServer("delivered-one", "wk-0054", "delivered", 1100),
				testServer("created-one", "wk-0054", "created", 1200),
			},
			wantMatch:      "created-one",
			wantCandidates: 2,
		},
		"two candidates in created status are ambiguous": {
			servers: []one_api.Server{
				testServer("a", "wk-0054", "created", 1100),
				testServer("b", "wk-0054", "created", 1200),
			},
			wantMatch:      "",
			wantCandidates: 2,
		},
		"two candidates without a created status are ambiguous": {
			servers: []one_api.Server{
				testServer("a", "wk-0054", "delivered", 1100),
				testServer("b", "wk-0054", "provisioning", 1200),
			},
			wantMatch:      "",
			wantCandidates: 2,
		},
		"stale duplicate outside the window does not create ambiguity": {
			servers: []one_api.Server{
				testServer("stale", "wk-0054", "delivered", 900),
				testServer("fresh", "wk-0054", "created", 1100),
			},
			wantMatch:      "fresh",
			wantCandidates: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			match, candidates := selectCreatedServer(tt.servers, criteria)

			gotMatch := ""
			if match != nil {
				gotMatch = match.Uuid
			}
			if gotMatch != tt.wantMatch {
				t.Errorf("match = %q, want %q", gotMatch, tt.wantMatch)
			}
			if len(candidates) != tt.wantCandidates {
				t.Errorf("candidates = %d, want %d", len(candidates), tt.wantCandidates)
			}
		})
	}
}

func TestRecoverCreatedServer(t *testing.T) {
	criteria := createdServerCriteria{
		Name:         "wk-0054",
		Location:     "EU: Rotterdam",
		InstanceType: "bm7.std.8",
		NotBefore:    1000,
	}
	found := testServer("a", "wk-0054", "created", 1100)

	listing := func(pages ...*one_api.ServerListResponse) (serverLister, *int) {
		calls := 0
		return func(context.Context) (*one_api.ServerListResponse, error) {
			calls++
			if calls > len(pages) {
				return pages[len(pages)-1], nil
			}
			return pages[calls-1], nil
		}, &calls
	}

	t.Run("adopts the server once it shows up", func(t *testing.T) {
		list, calls := listing(
			&one_api.ServerListResponse{},
			&one_api.ServerListResponse{Servers: []one_api.Server{found}},
		)

		match, _, err := recoverCreatedServer(context.Background(), list, criteria, time.Second, time.Millisecond)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if match == nil || match.Uuid != "a" {
			t.Fatalf("match = %+v, want server a", match)
		}
		if *calls < 2 {
			t.Errorf("calls = %d, want at least 2", *calls)
		}
	})

	t.Run("gives up when the server never shows up", func(t *testing.T) {
		list, _ := listing(&one_api.ServerListResponse{})

		match, candidates, err := recoverCreatedServer(context.Background(), list, criteria, 20*time.Millisecond, time.Millisecond)
		if !errors.Is(err, errCreatedServerNotFound) {
			t.Fatalf("err = %v, want errCreatedServerNotFound", err)
		}
		if match != nil || candidates != nil {
			t.Errorf("match = %+v, candidates = %+v, want none", match, candidates)
		}
	})

	t.Run("reports a lookup failure when the list never succeeds", func(t *testing.T) {
		list := func(context.Context) (*one_api.ServerListResponse, error) {
			return nil, errors.New("boom")
		}

		_, _, err := recoverCreatedServer(context.Background(), list, criteria, 20*time.Millisecond, time.Millisecond)
		if !errors.Is(err, errCreatedServerLookupFailed) {
			t.Fatalf("err = %v, want errCreatedServerLookupFailed", err)
		}
		if errors.Is(err, errCreatedServerNotFound) {
			t.Error("a failed lookup must not be reported as a server that was never registered")
		}
	})

	t.Run("stops immediately when candidates are ambiguous", func(t *testing.T) {
		list, calls := listing(&one_api.ServerListResponse{Servers: []one_api.Server{
			testServer("a", "wk-0054", "created", 1100),
			testServer("b", "wk-0054", "created", 1200),
		}})

		match, candidates, err := recoverCreatedServer(context.Background(), list, criteria, time.Second, time.Millisecond)
		if !errors.Is(err, errAmbiguousCreatedServers) {
			t.Fatalf("err = %v, want errAmbiguousCreatedServers", err)
		}
		if match != nil {
			t.Errorf("match = %+v, want nil", match)
		}
		if len(candidates) != 2 {
			t.Errorf("candidates = %d, want 2", len(candidates))
		}
		if *calls != 1 {
			t.Errorf("calls = %d, want 1: polling cannot resolve ambiguity", *calls)
		}
	})

	t.Run("keeps polling when a list call fails", func(t *testing.T) {
		calls := 0
		list := func(context.Context) (*one_api.ServerListResponse, error) {
			calls++
			if calls == 1 {
				return nil, errors.New("boom")
			}
			return &one_api.ServerListResponse{Servers: []one_api.Server{found}}, nil
		}

		match, _, err := recoverCreatedServer(context.Background(), list, criteria, time.Second, time.Millisecond)
		if err != nil {
			t.Fatalf("err = %v, want nil", err)
		}
		if match == nil || match.Uuid != "a" {
			t.Fatalf("match = %+v, want server a", match)
		}
	})

	t.Run("stops when the context is cancelled", func(t *testing.T) {
		list, _ := listing(&one_api.ServerListResponse{})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, _, err := recoverCreatedServer(ctx, list, criteria, time.Second, time.Millisecond)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	})
}

func TestCreateRecoveryPossible(t *testing.T) {
	transportErr := fmt.Errorf("failed to do HTTP request: %w: %w",
		one_api.ErrRequestNotCompleted, context.DeadlineExceeded)

	tests := map[string]struct {
		postErr error
		ctxErr  error
		want    bool
	}{
		"incomplete request with budget left": {postErr: transportErr, want: true},
		"incomplete request after the create timeout": {
			postErr: transportErr, ctxErr: context.DeadlineExceeded, want: false,
		},
		"incomplete request after cancellation": {
			postErr: transportErr, ctxErr: context.Canceled, want: false,
		},
		"request that reached the API": {postErr: errors.New("error decoding response"), want: false},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := createRecoveryPossible(tt.postErr, tt.ctxErr); got != tt.want {
				t.Errorf("createRecoveryPossible = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateRecoveryUnavailableHint(t *testing.T) {
	tests := map[string]struct {
		ctxErr      error
		wantPhrases []string
	}{
		"create timeout exhausted": {
			ctxErr:      context.DeadlineExceeded,
			wantPhrases: []string{"create timeout", "terraform import", "may have been registered"},
		},
		"apply cancelled": {
			ctxErr:      context.Canceled,
			wantPhrases: []string{"cancelled", "terraform import", "may have been registered"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := createRecoveryUnavailableHint(tt.ctxErr)
			for _, want := range tt.wantPhrases {
				if !strings.Contains(got, want) {
					t.Errorf("hint %q does not mention %q", got, want)
				}
			}
		})
	}
}
