package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type (
	Client struct {
		http *http.Client
	}

	Option func(*http.Client)

	Request struct {
		Method       string
		URL          string
		Query        url.Values
		Header       http.Header
		Bearer       string
		JSON         any
		Form         url.Values
		MaxBodyBytes int64
	}

	StatusError struct {
		StatusCode int
		Header     http.Header
		Body       []byte
	}
)

const (
	defaultMaxBodyBytes = 10 << 20
	maxErrorBodyBytes   = 4 << 10
	maxDrainBytes       = 64 << 10
)

var (
	ErrBodyConflict = errors.New("httpclient: a request carries either a JSON body or a form body, not both")
)

func New(timeout time.Duration, opts ...Option) *Client {
	client := &http.Client{Timeout: timeout}
	for _, opt := range opts {
		opt(client)
	}

	return &Client{http: client}
}

func WithTransport(transport http.RoundTripper) Option {
	return func(client *http.Client) {
		client.Transport = transport
	}
}

func WithMaxRedirects(n int) Option {
	return func(client *http.Client) {
		client.CheckRedirect = func(_ *http.Request, via []*http.Request) error {
			if len(via) > n {
				return http.ErrUseLastResponse
			}

			return nil
		}
	}
}

func (c *Client) HTTP() *http.Client {
	return c.http
}

func (c *Client) Do(ctx context.Context, req Request) (*http.Response, error) {
	httpReq, err := build(ctx, req)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, redactURL(err)
	}

	return resp, nil
}

func (c *Client) JSON(ctx context.Context, req Request, out any) error {
	resp, err := c.Do(ctx, req)
	if err != nil {
		return err
	}
	defer drainAndClose(resp.Body)

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return newStatusError(resp)
	}

	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	limit := req.MaxBodyBytes
	if limit <= 0 {
		limit = defaultMaxBodyBytes
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, limit)).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}

func (e *StatusError) Error() string {
	if len(e.Body) == 0 {
		return fmt.Sprintf("unexpected status %d", e.StatusCode)
	}

	return fmt.Sprintf("unexpected status %d: %s", e.StatusCode, e.Body)
}

func build(ctx context.Context, req Request) (*http.Request, error) {
	if req.JSON != nil && req.Form != nil {
		return nil, ErrBodyConflict
	}

	target, err := url.Parse(req.URL)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	if len(req.Query) > 0 {
		query := target.Query()
		for key, values := range req.Query {
			for _, value := range values {
				query.Add(key, value)
			}
		}
		target.RawQuery = query.Encode()
	}

	var (
		body        io.Reader
		contentType string
	)
	switch {
	case req.JSON != nil:
		raw, err := json.Marshal(req.JSON)
		if err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(raw)
		contentType = "application/json"
	case req.Form != nil:
		body = strings.NewReader(req.Form.Encode())
		contentType = "application/x-www-form-urlencoded"
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, target.String(), body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	if req.Header != nil {
		httpReq.Header = req.Header.Clone()
	}
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}
	if req.Bearer != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.Bearer)
	}

	return httpReq, nil
}

func newStatusError(resp *http.Response) *StatusError {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))

	return &StatusError{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       raw,
	}
}

func drainAndClose(body io.ReadCloser) {
	_, _ = io.Copy(io.Discard, io.LimitReader(body, maxDrainBytes))
	_ = body.Close()
}

func redactURL(err error) error {
	urlErr, ok := errors.AsType[*url.Error](err)
	if !ok {
		return err
	}

	target, parseErr := url.Parse(urlErr.URL)
	if parseErr != nil {
		return urlErr.Err
	}

	target.User = nil
	target.RawQuery = ""
	target.Fragment = ""

	return &url.Error{Op: urlErr.Op, URL: target.String(), Err: urlErr.Err}
}
