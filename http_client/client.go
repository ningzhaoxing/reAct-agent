package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

type HTTPMethod string

const (
	HTTPMethodGET  HTTPMethod = "GET"
	HTTPMethodPOST HTTPMethod = "POST"
)

type HTTPHeader map[string]string

type HTTPResponse struct {
	Body       []byte
	StatusCode int
}

type IOReader <-chan HTTPResponse
type IOError <-chan error

type IHTTPClient interface {
	Send(ctx context.Context, method HTTPMethod, body interface{}) (*HTTPResponse, error)
	SendStream(ctx context.Context, method HTTPMethod, body interface{}) (IOReader, IOError)
}

type HTTPClient struct {
	baseUrl string
	path    string
	header  *HTTPHeader
	timeout time.Duration
}

// Option defines a functional option to configure HTTPClient.
type Option func(*HTTPClient)

// WithHeader sets a custom header map.
func WithHeader(h HTTPHeader) Option {
	return func(c *HTTPClient) {
		if c == nil {
			return
		}
		if h == nil {
			return
		}
		c.header = &h
	}
}

// WithTimeout sets a custom timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *HTTPClient) {
		if c == nil {
			return
		}
		if d <= 0 {
			return
		}
		c.timeout = d
	}
}

// NewHTTPClient creates a new HTTPClient with provided values.
// If header is nil, a default JSON header is used. If timeout is 0, it defaults to 30s.
func NewHTTPClient(baseUrl, path string, opts ...Option) *HTTPClient {
	// defaults
	defaultHeader := HTTPHeader{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}
	c := &HTTPClient{
		baseUrl: baseUrl,
		path:    path,
		header:  &defaultHeader,
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	return c
}

// NewDefaultHTTPClient creates a client with empty baseUrl/path and sensible defaults.
func NewDefaultHTTPClient(opts ...Option) *HTTPClient {
	return NewHTTPClient("", "", opts...)
}

// buildURL constructs the full URL from baseUrl and path.
func (c *HTTPClient) buildURL() string {
	if c == nil {
		return ""
	}
	base := c.baseUrl
	p := c.path
	if base == "" {
		return p
	}
	if p == "" {
		return base
	}
	// ensure single slash between base and path
	if base[len(base)-1] == '/' && p[0] == '/' {
		return base + p[1:]
	}
	if base[len(base)-1] != '/' && p[0] != '/' {
		return base + "/" + p
	}
	return base + p
}

// Ensure HTTPClient implements IHTTPClient
var _ IHTTPClient = (*HTTPClient)(nil)

// Send performs a simple HTTP request and returns the whole response body.
func (c *HTTPClient) Send(ctx context.Context, method HTTPMethod, body interface{}) (*HTTPResponse, error) {
	url := c.buildURL()
	// prepare body reader
	var reader io.Reader
	switch v := body.(type) {
	case nil:
		reader = nil
	case []byte:
		reader = bytes.NewReader(v)
	case string:
		reader = bytes.NewBufferString(v)
	default:
		// marshal to JSON by default
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, string(method), url, reader)
	if err != nil {
		return nil, err
	}
	// apply headers
	if c.header != nil {
		for k, v := range *c.header {
			req.Header.Set(k, v)
		}
	}

	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &HTTPResponse{Body: b, StatusCode: resp.StatusCode}, nil
}

// SendStream performs the request and streams the response body in chunks.
func (c *HTTPClient) SendStream(ctx context.Context, method HTTPMethod, body interface{}) (IOReader, IOError) {
	out := make(chan HTTPResponse)
	errs := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errs)

		url := c.buildURL()
		// prepare body reader
		var reader io.Reader
		switch v := body.(type) {
		case nil:
			reader = nil
		case []byte:
			reader = bytes.NewReader(v)
		case string:
			reader = bytes.NewBufferString(v)
		default:
			b, err := json.Marshal(v)
			if err != nil {
				errs <- err
				return
			}
			reader = bytes.NewReader(b)
		}

		req, err := http.NewRequestWithContext(ctx, string(method), url, reader)
		if err != nil {
			errs <- err
			return
		}
		if c.header != nil {
			for k, v := range *c.header {
				req.Header.Set(k, v)
			}
		}

		client := &http.Client{Timeout: c.timeout}
		resp, err := client.Do(req)
		if err != nil {
			errs <- err
			return
		}
		defer resp.Body.Close()

		buf := make([]byte, 8*1024)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				// copy the buffer chunk to avoid data race
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				out <- HTTPResponse{Body: chunk}
			}
			if err != nil {
				if err == io.EOF {
					return
				}
				errs <- err
				return
			}
		}
	}()

	return out, errs
}
