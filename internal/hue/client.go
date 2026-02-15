package hue

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
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
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Group     string   `json:"group,omitempty"`
	GroupType string   `json:"group_type,omitempty"`
	On        bool     `json:"on"`
	Reachable *bool    `json:"reachable,omitempty"`
	Bri       *float64 `json:"bri,omitempty"`
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

type groupedLightsResponse struct {
	Data   []groupedLightResource `json:"data"`
	Errors []apiErrorDetail       `json:"errors"`
}

type groupsResponse struct {
	Data   []groupResource  `json:"data"`
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
	ID    string `json:"id"`
	Owner *struct {
		RID   string `json:"rid"`
		RType string `json:"rtype"`
	} `json:"owner,omitempty"`
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	On struct {
		On bool `json:"on"`
	} `json:"on"`
	Dimming *struct {
		Brightness float64 `json:"brightness"`
	} `json:"dimming,omitempty"`
	Status string `json:"status,omitempty"`
}

type groupedLightResource struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Owner struct {
		RID   string `json:"rid"`
		RType string `json:"rtype"`
	} `json:"owner"`
}

type groupResource struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Services []resourceRef `json:"services"`
	Children []resourceRef `json:"children"`
}

type resourceRef struct {
	RID   string `json:"rid"`
	RType string `json:"rtype"`
}

type ListLightsOptions struct {
	WithGroup bool
	WithState bool
}

type lightGroupInfo struct {
	Name string
	Type string
}

func NewClient(cfg config.Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if cfg.InsecureTLS {
		httpClient = cloneWithInsecureTLS(httpClient)
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
	return c.ListLightsWithOptions(ctx, ListLightsOptions{})
}

func (c *Client) ListLightsWithOptions(ctx context.Context, opts ListLightsOptions) ([]Light, error) {
	var resp listLightsResponse
	if err := c.doJSON(ctx, http.MethodGet, apiBasePath+"/resource/light", nil, &resp); err != nil {
		return nil, err
	}

	var lightToGroup map[string]lightGroupInfo
	if opts.WithGroup {
		var err error
		lightToGroup, err = c.fetchLightGroupMap(ctx, resp.Data)
		if err != nil {
			return nil, err
		}
	}

	lights := make([]Light, 0, len(resp.Data))
	for _, item := range resp.Data {
		light := Light{
			ID:   item.ID,
			Name: item.Metadata.Name,
			On:   item.On.On,
		}
		if opts.WithState {
			if item.Dimming != nil {
				bri := item.Dimming.Brightness
				light.Bri = &bri
			}
			if item.Status != "" {
				reachable := strings.EqualFold(item.Status, "connected")
				light.Reachable = &reachable
			}
		}
		if opts.WithGroup {
			if group, ok := lightToGroup[item.ID]; ok {
				light.Group = group.Name
				light.GroupType = group.Type
			}
		}
		lights = append(lights, light)
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
	case *groupedLightsResponse:
		if len(typed.Errors) > 0 {
			return fmt.Errorf("hue api error: %s", typed.Errors[0].Description)
		}
	case *groupsResponse:
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

func cloneWithInsecureTLS(httpClient *http.Client) *http.Client {
	clonedClient := *httpClient

	var transport *http.Transport
	switch t := clonedClient.Transport.(type) {
	case nil:
		transport = http.DefaultTransport.(*http.Transport).Clone()
	case *http.Transport:
		transport = t.Clone()
	default:
		transport = http.DefaultTransport.(*http.Transport).Clone()
	}

	if transport.TLSClientConfig == nil {
		transport.TLSClientConfig = &tls.Config{}
	}
	transport.TLSClientConfig.InsecureSkipVerify = true
	clonedClient.Transport = transport

	return &clonedClient
}

func (c *Client) fetchLightGroupMap(ctx context.Context, lights []lightResource) (map[string]lightGroupInfo, error) {
	groupedLights, err := c.fetchGroupedLights(ctx)
	if err != nil {
		return nil, err
	}
	groups, err := c.fetchGroups(ctx)
	if err != nil {
		return nil, err
	}

	groupedByOwner := make(map[string]groupedLightResource, len(groupedLights))
	for _, gl := range groupedLights {
		groupedByOwner[gl.Owner.RID] = gl
	}
	ownerToLights := make(map[string][]string)
	for _, light := range lights {
		if light.Owner == nil || light.Owner.RType != "device" {
			continue
		}
		ownerToLights[light.Owner.RID] = append(ownerToLights[light.Owner.RID], light.ID)
	}

	infoByGroupedID := make(map[string]lightGroupInfo, len(groupedLights))
	for _, gl := range groupedLights {
		infoByGroupedID[gl.ID] = lightGroupInfo{Name: "", Type: gl.Type}
	}
	for _, group := range groups {
		groupedID := firstGroupedLightID(group.Services)
		if groupedID == "" {
			if gl, ok := groupedByOwner[group.ID]; ok {
				groupedID = gl.ID
			}
		}
		if groupedID == "" {
			continue
		}
		infoByGroupedID[groupedID] = lightGroupInfo{
			Name: group.Metadata.Name,
			Type: group.Type,
		}
	}

	lightToGroup := make(map[string]lightGroupInfo)
	for _, group := range groups {
		groupedID := firstGroupedLightID(group.Services)
		if groupedID == "" {
			if gl, ok := groupedByOwner[group.ID]; ok {
				groupedID = gl.ID
			}
		}
		if groupedID == "" {
			continue
		}
		info, ok := infoByGroupedID[groupedID]
		if !ok {
			continue
		}
		for _, child := range group.Children {
			switch child.RType {
			case "light":
				// Deterministic on duplicates: first group in sorted order wins.
				if _, exists := lightToGroup[child.RID]; !exists {
					lightToGroup[child.RID] = info
				}
			case "device":
				for _, lightID := range ownerToLights[child.RID] {
					if _, exists := lightToGroup[lightID]; !exists {
						lightToGroup[lightID] = info
					}
				}
			}
		}
	}

	return lightToGroup, nil
}

func (c *Client) fetchGroupedLights(ctx context.Context) ([]groupedLightResource, error) {
	var resp groupedLightsResponse
	if err := c.doJSON(ctx, http.MethodGet, apiBasePath+"/resource/grouped_light", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *Client) fetchGroups(ctx context.Context) ([]groupResource, error) {
	var roomsResp groupsResponse
	if err := c.doJSON(ctx, http.MethodGet, apiBasePath+"/resource/room", nil, &roomsResp); err != nil {
		return nil, err
	}
	var zonesResp groupsResponse
	if err := c.doJSON(ctx, http.MethodGet, apiBasePath+"/resource/zone", nil, &zonesResp); err != nil {
		return nil, err
	}

	groups := append([]groupResource{}, roomsResp.Data...)
	groups = append(groups, zonesResp.Data...)
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Type == groups[j].Type {
			return groups[i].ID < groups[j].ID
		}
		return groups[i].Type < groups[j].Type
	})
	return groups, nil
}

func firstGroupedLightID(services []resourceRef) string {
	for _, service := range services {
		if service.RType == "grouped_light" {
			return service.RID
		}
	}
	return ""
}

func FormatBrightness(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(int(*value + 0.5))
}
