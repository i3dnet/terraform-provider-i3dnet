package one_api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"
)

func TestLoggingRoundTripperRedactsTheAPIKey(t *testing.T) {
	var gotKey, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("PRIVATE-TOKEN")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var logs bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &logs)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL, strings.NewReader(`{"name":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("PRIVATE-TOKEN", "secret-api-key")

	resp, err := (&http.Client{Transport: &loggingRoundTripper{next: http.DefaultTransport, ctx: ctx}}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if strings.Contains(logs.String(), "secret-api-key") {
		t.Errorf("logs contain the API key:\n%s", logs.String())
	}
	if !strings.Contains(logs.String(), `Private-Token: ***`) {
		t.Errorf("logs do not show the redacted header:\n%s", logs.String())
	}
	if !strings.Contains(logs.String(), `{\"name\":\"x\"}`) {
		t.Errorf("logs do not contain the request body:\n%s", logs.String())
	}
	if gotKey != "secret-api-key" {
		t.Errorf("server got PRIVATE-TOKEN %q, want the real key", gotKey)
	}
	if gotBody != `{"name":"x"}` {
		t.Errorf("server got body %q", gotBody)
	}
	if req.Header.Get("PRIVATE-TOKEN") != "secret-api-key" {
		t.Errorf("request headers were left redacted")
	}
}
