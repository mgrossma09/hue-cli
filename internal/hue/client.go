package hue

import (
	"context"
	"errors"
	"net/http"

	"github.com/mgrossma09/hue-cli/internal/config"
)

var ErrNotImplemented = errors.New("hue client methods are not implemented in Phase 1")

type Client struct {
	BridgeHost string
	APIToken   string
	HTTPClient *http.Client
}

type Light struct {
	ID   string
	Name string
	On   bool
}

type UpdateLightRequest struct {
	On  *bool
	Bri *int
	CT  *int
	XY  *XY
}

type XY struct {
	X float64
	Y float64
}

func NewClient(cfg config.Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		BridgeHost: cfg.BridgeHost,
		APIToken:   cfg.APIToken,
		HTTPClient: httpClient,
	}
}

func (c *Client) ListLights(ctx context.Context) ([]Light, error) {
	_ = ctx
	return nil, ErrNotImplemented
}

func (c *Client) ToggleLight(ctx context.Context, id string) error {
	_, _ = ctx, id
	return ErrNotImplemented
}

func (c *Client) UpdateLight(ctx context.Context, id string, req UpdateLightRequest) error {
	_, _, _ = ctx, id, req
	return ErrNotImplemented
}
