package cdn_api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Resource is a CDN resource as returned by GET /service/resource/{uid}.
type Resource struct {
	UID            string `json:"uid"`
	Name           string `json:"name"`
	GroupCode      string `json:"groupCode"`
	InitiativeCode string `json:"initiativeCode"`
}

// CreateResourceRequest is the body of POST /service/resource.
type CreateResourceRequest struct {
	Name           string `json:"name"`
	GroupCode      string `json:"groupCode"`
	InitiativeCode string `json:"initiativeCode"`
}

// Healthcheck is how the edges probe an origin.
type Healthcheck struct {
	Host               string `json:"host"`
	Path               string `json:"path"`
	ExpectedStatusCode int64  `json:"expected_status_code"`
	TimeoutMs          int64  `json:"timeout_ms"`
	IntervalMs         int64  `json:"interval_ms"`
	MinValidChecks     int64  `json:"min_valid_checks"`
}

// Origin is one server the edges fetch content from.
type Origin struct {
	FQDN        string       `json:"fqdn"`
	Port        int64        `json:"port"`
	SSL         bool         `json:"ssl"`
	VerifySSL   bool         `json:"verify_ssl"`
	Location    string       `json:"location"`
	DNSRecord   string       `json:"dns_record"`
	Healthcheck *Healthcheck `json:"healthcheck"`
}

// Backend is the origin selection strategy and the origins.
type Backend struct {
	Strategy string   `json:"strategy"`
	Origins  []Origin `json:"origin_list"`
}

// Hostname is one public hostname of a resource.
type Hostname struct {
	FQDN  string `json:"fqdn"`
	SSLCN string `json:"ssl_cn"`
}

// Flags are the resource's on/off settings.
type Flags struct {
	Enabled             bool `json:"enabled"`
	ForceHTTPS          bool `json:"force_https"`
	AuthPassThrough     bool `json:"auth_pass_through"`
	OverrideBackendHost bool `json:"override_backend_host"`
	OverrideHealthHost  bool `json:"override_health_host"`
	UseI3DStaticDNS     bool `json:"use_i3d_static_dns"`
}

// PredefinedAllowList is a predefined allowlist, by key.
type PredefinedAllowList struct {
	Key         string `json:"key"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
}

// Every setting the API keeps per resource. DELETE /service/resource/{uid} is
// refused while any of them is still set, including settings this provider does
// not manage (they may have been made in the CDN dashboard).
var resourceSettings = []string{
	"flags",
	"hostnames",
	"backend",
	"redirect",
	"rewritelist",
	"customHeaders",
	"connectionthrottle",
	"pathdenylist",
	"geodenylist",
	"denylist",
	"tokentype",
	"allowlist",
	"predefined-allowlist",
}

func resourcePath(uid string) string {
	return "service/resource/" + url.PathEscape(uid)
}

// CreateResource creates a resource and returns its uid.
func (c *Client) CreateResource(ctx context.Context, req CreateResourceRequest) (string, error) {
	var out Resource
	if err := c.do(ctx, http.MethodPost, "service/resource", req, &out); err != nil {
		return "", err
	}

	if out.UID == "" {
		return "", errors.New("POST service/resource: the response carries no uid")
	}

	return out.UID, nil
}

// GetResource reads a resource. A missing resource is an error for which
// IsNotFound is true.
func (c *Client) GetResource(ctx context.Context, uid string) (*Resource, error) {
	var out Resource
	if err := c.do(ctx, http.MethodGet, resourcePath(uid), nil, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// RenameResource changes only the resource's name.
func (c *Client) RenameResource(ctx context.Context, uid, name string) error {
	return c.do(ctx, http.MethodPut, resourcePath(uid), map[string]string{"name": name}, nil)
}

// GetBackend reads the backend. An unset backend has no strategy and no origins.
func (c *Client) GetBackend(ctx context.Context, uid string) (*Backend, error) {
	var out Backend
	if err := c.do(ctx, http.MethodGet, resourcePath(uid)+"/backend", nil, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// PutBackend replaces the backend.
func (c *Client) PutBackend(ctx context.Context, uid string, b Backend) error {
	return c.do(ctx, http.MethodPut, resourcePath(uid)+"/backend", b, nil)
}

// GetHostnames reads the hostnames.
func (c *Client) GetHostnames(ctx context.Context, uid string) ([]Hostname, error) {
	var out []Hostname
	if err := c.do(ctx, http.MethodGet, resourcePath(uid)+"/hostnames", nil, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// PutHostnames replaces the hostnames.
func (c *Client) PutHostnames(ctx context.Context, uid string, h []Hostname) error {
	return c.do(ctx, http.MethodPut, resourcePath(uid)+"/hostnames", h, nil)
}

// GetFlags reads the flags.
func (c *Client) GetFlags(ctx context.Context, uid string) (*Flags, error) {
	var out Flags
	if err := c.do(ctx, http.MethodGet, resourcePath(uid)+"/flags", nil, &out); err != nil {
		return nil, err
	}

	return &out, nil
}

// PutFlags replaces the flags.
func (c *Client) PutFlags(ctx context.Context, uid string, f Flags) error {
	return c.do(ctx, http.MethodPut, resourcePath(uid)+"/flags", f, nil)
}

// GetPredefinedAllowList returns the key of the resource's predefined
// allowlist, or "" when none is set.
func (c *Client) GetPredefinedAllowList(ctx context.Context, uid string) (string, error) {
	var out PredefinedAllowList
	if err := c.do(ctx, http.MethodGet, resourcePath(uid)+"/predefined-allowlist", nil, &out); err != nil {
		return "", err
	}

	return out.Key, nil
}

// PutPredefinedAllowList selects a predefined allowlist.
func (c *Client) PutPredefinedAllowList(ctx context.Context, uid, key string) error {
	return c.do(ctx, http.MethodPut, resourcePath(uid)+"/predefined-allowlist", PredefinedAllowList{Key: key}, nil)
}

// DeleteSetting clears one setting of the resource (for example "flags").
// Clearing a setting that is not set succeeds.
func (c *Client) DeleteSetting(ctx context.Context, uid, setting string) error {
	return c.do(ctx, http.MethodDelete, resourcePath(uid)+"/"+setting, nil, nil)
}

// ListPredefinedAllowLists returns the predefined allowlists a resource can use.
func (c *Client) ListPredefinedAllowLists(ctx context.Context) ([]PredefinedAllowList, error) {
	var out []PredefinedAllowList
	if err := c.do(ctx, http.MethodGet, "service/acl/predefined", nil, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// DeleteResource clears every setting of the resource, then deletes it.
//
// The API checks that every setting is cleared before it deletes a resource, but
// it may not yet see a clear it has just accepted. A refusal of that kind is
// retried every retryInterval until ctx expires. A resource that is already gone
// is not an error.
func (c *Client) DeleteResource(ctx context.Context, uid string, retryInterval time.Duration) error {
	for _, s := range resourceSettings {
		if err := c.DeleteSetting(ctx, uid, s); err != nil {
			if IsNotFound(err) {
				continue
			}

			return fmt.Errorf("clearing %s: %w", s, err)
		}
	}

	for {
		err := c.do(ctx, http.MethodDelete, resourcePath(uid), nil, nil)
		if err == nil {
			return nil
		}

		if _, getErr := c.GetResource(ctx, uid); IsNotFound(getErr) {
			return nil
		}

		if !isSettingsStillSet(err) {
			return err
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("%w (gave up waiting: %w)", err, ctx.Err())
		case <-time.After(retryInterval):
		}
	}
}

// isSettingsStillSet reports whether err is the API refusing to delete a
// resource because a setting is still set.
func isSettingsStillSet(err error) bool {
	var apiErr *APIError

	return errors.As(err, &apiErr) &&
		apiErr.StatusCode == http.StatusBadRequest &&
		strings.Contains(apiErr.Message, "still has")
}
