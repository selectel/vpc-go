package v2

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/selectel/vpc-go/internal/utils"
)

const (
	authTokenHeader = "X-Auth-Token"
	userAgentHeader = "User-Agent"
)

var moduleUserAgent = "vpc-go/" + utils.ModuleVersion

// HTTPClient is the transport used by Client.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Config contains the caller-owned network API access scope.
type Config struct {
	Endpoint   string
	Token      string
	UserAgent  string
	HTTPClient HTTPClient
}

// Client sends requests to a caller-provided network API endpoint.
type Client struct {
	endpoint   *url.URL
	token      string
	userAgent  string
	httpClient HTTPClient
}

// NewClient creates a network API client without performing authentication.
func NewClient(config Config) (*Client, error) {
	if strings.TrimSpace(config.Endpoint) == "" {
		return nil, errors.New("endpoint is required")
	}
	if config.Token == "" {
		return nil, errors.New("token is required")
	}

	endpoint, err := url.Parse(config.Endpoint)
	if err != nil {
		return nil, err
	}
	if endpoint.Scheme == "" || endpoint.Host == "" {
		return nil, errors.New("endpoint must be an absolute URL")
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 120 * time.Second}
	}

	userAgent := moduleUserAgent
	if config.UserAgent != "" {
		userAgent = config.UserAgent + " " + moduleUserAgent
	}

	return &Client{
		endpoint:   endpoint,
		token:      config.Token,
		userAgent:  userAgent,
		httpClient: httpClient,
	}, nil
}

// Do sends exactly one request through the configured HTTP client.
//
// Resource packages use path relative to the configured endpoint. The caller
// owns response body closure.
func (client *Client) Do(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	body io.Reader,
) (*http.Response, error) {
	requestURL := client.endpoint.JoinPath(strings.TrimPrefix(path, "/"))
	if len(query) != 0 {
		requestURL.RawQuery = query.Encode()
	}

	request, err := http.NewRequestWithContext(ctx, method, requestURL.String(), body)
	if err != nil {
		return nil, newRequestError(err)
	}

	request.Header.Set(authTokenHeader, client.token)
	request.Header.Set(userAgentHeader, client.userAgent)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return nil, &TransportError{Err: err}
	}

	return response, nil
}
