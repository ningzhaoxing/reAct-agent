package chatmodel

import (
	"errors"
	"net/http"
	"time"
)

type QWenModelClient struct {
	BaseUrl   string
	AuthToken string
	Timeout   time.Duration
	Path      string // default: chat/completions

	HTTPClient *http.Client
}

type Option func(*QWenModelClient) error

func NewQWenModelClient(authToken string, opts ...Option) (*QWenModelClient, error) {
	if authToken == "" {
		return nil, errors.New("authToken is required")
	}

	client := &QWenModelClient{
		AuthToken: authToken,
		Timeout:   5 * time.Minute,
		Path:      "chat/completions",
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}

	for _, opt := range opts {
		if err := opt(client); err != nil {
			return nil, err
		}
	}

	return client, nil
}
