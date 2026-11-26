package one_api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// loggingRoundTripper logs One API request and response if TF_LOG=DEBUG
// see: https://developer.hashicorp.com/terraform/plugin/log/writing
type loggingRoundTripper struct {
	next http.RoundTripper
	ctx  context.Context
}

func (l *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	reqBody, err := dumpRequestRedacted(req)
	if err != nil {
		return nil, fmt.Errorf("error dumping request: %w", err)
	}

	tflog.Debug(l.ctx, "Sending api request", map[string]interface{}{"body": string(reqBody), "method": req.Method})

	resp, err := l.next.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	respBody, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return nil, fmt.Errorf("error dumping response: %w", err)
	}

	tflog.Debug(l.ctx, "Getting api response", map[string]interface{}{"body": string(respBody), "duration": time.Since(start)})

	return resp, nil
}

// redactedHeaders hold credentials and are never written to the logs.
var redactedHeaders = []string{"PRIVATE-TOKEN", "Authorization"}

// dumpRequestRedacted dumps req like httputil.DumpRequest, with the values of
// redactedHeaders replaced. req itself is sent with its headers unchanged.
func dumpRequestRedacted(req *http.Request) ([]byte, error) {
	original := req.Header
	redacted := original.Clone()
	for _, h := range redactedHeaders {
		if redacted.Get(h) != "" {
			redacted.Set(h, "***")
		}
	}

	req.Header = redacted
	defer func() { req.Header = original }()

	return httputil.DumpRequest(req, true)
}
