package hue

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

	"github.com/mgrossma09/hue-cli/internal/config"
)

const apiBasePath = "/clip/v2"

var (
	ErrNoUpdateFields = errors.New("no update fields provided")
	ErrMissingID      = errors.New("light id is required")
	ErrLightNotFound  = errors.New("light not found")
)

type Client struct {
	BridgeHost string
	APIToken   string
	BaseURL    string
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

type apiErrorDetail struct {
	Description string `json:"description"`
}

type listLightsResponse struct {
	Data   []lightResource  `json:"data"`
	Errors []apiErrorDetail `json:"errors"`
}

type lightResponse struct {
	Data   []lightResource  `json:"data"`
	Errors []apiErrorDetail `json:"errors"`
}

type updateResponse struct {
	Errors []apiErrorDetail `json:"errors"`
}

type lightResource struct {
	ID       string `json:"id"`
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	On struct {
		On bool `json:"on"`
	} `json:"on"`
}

func NewClient(cfg config.Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	baseURL := normalizeBaseURL(cfg.BridgeHost)
	return &Client{
		BridgeHost: cfg.BridgeHost,
		APIToken:   cfg.APIToken,
		BaseURL:    baseURL,
		HTTPClient: httpClient,
	}
}

func (c *Client) ListLights(ctx context.Context) ([]Light, error) {
	var resp listLightsResponse
	if err := c.doJSON(ctx, http.MethodGet, apiBasePath+"/resource/light", nil, &resp); err != nil {
		return nil, err
	}

	lights := make([]Light, 0, len(resp.Data))
	for _, item := range resp.Data {
		lights = append(lights, Light{
			ID:   item.ID,
			Name: item.Metadata.Name,
			On:   item.On.On,
		})
	}
	return lights, nil
}

func (c *Client) ToggleLight(ctx context.Context, id string) error {
	if id == "" {
		return ErrMissingID
	}

	light, err := c.GetLight(ctx, id)
	if err != nil {
		return err
	}

	nextState := !light.On
	return c.UpdateLight(ctx, id, UpdateLightRequest{On: &nextState})
}

func (c *Client) GetLight(ctx context.Context, id string) (Light, error) {
	if id == "" {
		return Light{}, ErrMissingID
	}

	var resp lightResponse
	path := apiBasePath + "/resource/light/" + url.PathEscape(id)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return Light{}, err
	}
	if len(resp.Data) == 0 {
		return Light{}, ErrLightNotFound
	}

	item := resp.Data[0]
	return Light{ID: item.ID, Name: item.Metadata.Name, On: item.On.On}, nil
}

func (c *Client) UpdateLight(ctx context.Context, id string, req UpdateLightRequest) error {
	if id == "" {
		return ErrMissingID
	}

	payload := map[string]any{}
	if req.On != nil {
		payload["on"] = map[string]bool{"on": *req.On}
	}
	if req.Bri != nil {
		payload["dimming"] = map[string]float64{"brightness": float64(*req.Bri)}
	}
	if req.CT != nil {
		payload["color_temperature"] = map[string]int{"mirek": *req.CT}
	}
	if req.XY != nil {
		payload["color"] = map[string]any{
			"xy": map[string]float64{
				"x": req.XY.X,
				"y": req.XY.Y,
			},
		}
	}
	if len(payload) == 0 {
		return ErrNoUpdateFields
	}

	var resp updateResponse
	path := apiBasePath + "/resource/light/" + url.PathEscape(id)
	if err := c.doJSON(ctx, http.MethodPut, path, payload, &resp); err != nil {
		return err
	}

	return nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, payload any, dest any) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("hue-application-key", c.APIToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("hue api status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	if len(respBody) == 0 || dest == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, dest); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	switch typed := dest.(type) {
	case *listLightsResponse:
		if len(typed.Errors) > 0 {
			return fmt.Errorf("hue api error: %s", typed.Errors[0].Description)
		}
	case *lightResponse:
		if len(typed.Errors) > 0 {
			return fmt.Errorf("hue api error: %s", typed.Errors[0].Description)
		}
	case *updateResponse:
		if len(typed.Errors) > 0 {
			return fmt.Errorf("hue api error: %s", typed.Errors[0].Description)
		}
	}

	return nil
}

func normalizeBaseURL(host string) string {
	trimmed := strings.TrimSpace(host)
	trimmed = strings.TrimSuffix(trimmed, "/")
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed
	}
	return "https://" + trimmed
}
