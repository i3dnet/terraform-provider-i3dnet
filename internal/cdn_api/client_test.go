package cdn_api

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"
)

func TestRequestsCarryTheKeyAsABearerTokenAndLogsRedactIt(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"uid":"abc","name":"n"}`))
	}))
	t.Cleanup(srv.Close)

	var logs bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &logs)

	c, err := NewClient("cdn_secret", srv.URL+"/api")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := c.GetResource(ctx, "abc"); err != nil {
		t.Fatal(err)
	}

	if gotAuth != "Bearer cdn_secret" {
		t.Errorf("Authorization = %q, want the bearer key", gotAuth)
	}
	if strings.Contains(logs.String(), "cdn_secret") {
		t.Error("the API key appears in the debug log")
	}
	if !strings.Contains(logs.String(), "Sending CDN API request") {
		t.Error("the request was not logged")
	}
}

func TestAnyTwoHundredIsSuccessAndErrorsCarryTheAPIMessage(t *testing.T) {
	status := http.StatusCreated
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		if status >= 400 {
			_, _ = w.Write([]byte(`{"code":404,"message":"no vhost found with the provided uid"}`))
		}
	}))
	t.Cleanup(srv.Close)

	c, _ := NewClient("k", srv.URL)

	if err := c.PutFlags(context.Background(), "abc", Flags{}); err != nil {
		t.Errorf("PUT answered 201: %v, want success", err)
	}

	status = http.StatusNotFound
	_, err := c.GetResource(context.Background(), "abc")
	if !IsNotFound(err) {
		t.Fatalf("err = %v, want not found", err)
	}
	if !strings.Contains(err.Error(), "no vhost found") {
		t.Errorf("err = %v, want the API's message", err)
	}
}

func TestDeleteResourceClearsEverySettingFirstAndRetriesWhileTheAPICatchesUp(t *testing.T) {
	var deletes []string
	refusals := 2
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deletes = append(deletes, r.Method+" "+r.URL.Path)
		if r.URL.Path == "/service/resource/abc" && r.Method == http.MethodDelete && refusals > 0 {
			refusals--
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":400,"message":"cannot delete resource because it still has flags entries associated to it"}`))
			return
		}
		if r.Method == http.MethodGet {
			_, _ = w.Write([]byte(`{"uid":"abc","name":"n"}`))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	c, _ := NewClient("k", srv.URL)
	if err := c.DeleteResource(context.Background(), "abc", time.Millisecond); err != nil {
		t.Fatal(err)
	}

	var settingDeletes int
	for _, d := range deletes {
		if strings.HasPrefix(d, "DELETE /service/resource/abc/") {
			settingDeletes++
		}
	}
	if settingDeletes != len(resourceSettings) {
		t.Errorf("cleared %d settings, want all %d", settingDeletes, len(resourceSettings))
	}
	if last := deletes[len(deletes)-1]; last != "DELETE /service/resource/abc" {
		t.Errorf("last call = %q, want the resource DELETE", last)
	}
}

func TestDeleteResourceGivesUpWhenTheContextExpires(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"uid":"abc","name":"n"}`))
		case r.URL.Path == "/service/resource/abc":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"code":400,"message":"cannot delete resource because it still has backend entries associated to it"}`))
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	c, _ := NewClient("k", srv.URL)
	err := c.DeleteResource(ctx, "abc", 5*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want the deadline", err)
	}
}

func TestATransportFailureIsErrNoResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close()

	c, _ := NewClient("k", srv.URL)
	_, err := c.CreateResource(context.Background(), CreateResourceRequest{Name: "n"})
	if !errors.Is(err, ErrNoResponse) {
		t.Fatalf("err = %v, want ErrNoResponse", err)
	}
}
