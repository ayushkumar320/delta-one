package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client calls another service's JSON API.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// NewClient returns a Client for a service base URL such as
// "http://localhost:8082". The timeout is deliberately short: a service
// waiting on a slow dependency should fail rather than pile up requests.
func NewClient(baseURL string) *Client {
	return &Client{BaseURL: baseURL, HTTP: &http.Client{Timeout: 5 * time.Second}}
}

// Do sends body (when not nil) to path and decodes the response into out
// (when not nil). Headers carrying the request ID and caller identity are
// copied from ctx by the caller through WithHeaders.
func (c *Client) Do(ctx context.Context, method, path string, body, out any, headers map[string]string) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("call %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return decodeError(resp)
	}
	if out == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s %s response: %w", method, path, err)
	}
	return nil
}

// decodeError turns a downstream error response back into an *Error, so a
// 404 from catalog surfaces to the user as a 404 rather than a 500.
func decodeError(resp *http.Response) error {
	var body errorBody
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&body); err != nil || body.Error.Code == "" {
		return NewError(resp.StatusCode, "upstream_error",
			fmt.Sprintf("upstream service returned %d", resp.StatusCode))
	}
	return NewError(resp.StatusCode, body.Error.Code, body.Error.Message)
}
