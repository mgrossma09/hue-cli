package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/mgrossma09/hue-cli/internal/config"
	"github.com/mgrossma09/hue-cli/internal/hue"
)

func main() {
	cfg, err := config.LoadAndValidate("")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	client := hue.NewClient(cfg, nil)

	s := server.NewMCPServer("hue", "0.1.0",
		server.WithToolCapabilities(true),
	)

	s.AddTool(buildListLightsTool(), makeListLightsHandler(client))
	s.AddTool(buildListRoomsTool(), makeListRoomsHandler(client))
	s.AddTool(buildToggleLightTool(), makeToggleLightHandler(client))
	s.AddTool(buildSetLightTool(), makeSetLightHandler(client))

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

// ── Tool definitions ──────────────────────────────────────────────────────────

func buildListLightsTool() mcp.Tool {
	return mcp.NewTool("list_lights",
		mcp.WithDescription("List all Hue lights, optionally filtered by room/zone name."),
		mcp.WithString("group",
			mcp.Description("Optional room or zone name to filter by (case-insensitive)."),
		),
	)
}

func buildListRoomsTool() mcp.Tool {
	return mcp.NewTool("list_rooms",
		mcp.WithDescription("List all unique room/zone names from the Hue bridge."),
	)
}

func buildToggleLightTool() mcp.Tool {
	return mcp.NewTool("toggle_light",
		mcp.WithDescription("Toggle a light on/off. Target by id, or by group (room/zone) with optional name filter."),
		mcp.WithString("id",
			mcp.Description("Light ID to toggle directly."),
		),
		mcp.WithString("group",
			mcp.Description("Room or zone name to target (case-insensitive)."),
		),
		mcp.WithString("name",
			mcp.Description("Light name filter within the group (case-insensitive)."),
		),
	)
}

func buildSetLightTool() mcp.Tool {
	return mcp.NewTool("set_light",
		mcp.WithDescription("Set light state. Target by id, or by group with optional name filter."),
		mcp.WithString("id",
			mcp.Description("Light ID to update directly."),
		),
		mcp.WithString("group",
			mcp.Description("Room or zone name to target (case-insensitive)."),
		),
		mcp.WithString("name",
			mcp.Description("Light name filter within the group (case-insensitive)."),
		),
		mcp.WithBoolean("on",
			mcp.Description("Turn light on (true) or off (false)."),
		),
		mcp.WithNumber("brightness",
			mcp.Description("Brightness percentage 0–100."),
		),
		mcp.WithNumber("color_temp",
			mcp.Description("Color temperature in mirek (153–500)."),
		),
	)
}

// ── Handlers ──────────────────────────────────────────────────────────────────

func makeListLightsHandler(client *hue.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		group := req.GetString("group", "")

		lights, err := client.ListLightsWithOptions(ctx, hue.ListLightsOptions{
			WithGroup: true,
			WithState: true,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list lights: %v", err)), nil
		}

		if group != "" {
			filtered := lights[:0]
			for _, l := range lights {
				if strings.EqualFold(l.Group, group) {
					filtered = append(filtered, l)
				}
			}
			lights = filtered
		}

		data, err := json.Marshal(lights)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("encode response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeListRoomsHandler(client *hue.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		lights, err := client.ListLightsWithOptions(ctx, hue.ListLightsOptions{WithGroup: true})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("list lights: %v", err)), nil
		}

		seen := make(map[string]bool)
		var rooms []string
		for _, l := range lights {
			if l.Group != "" && !seen[l.Group] {
				seen[l.Group] = true
				rooms = append(rooms, l.Group)
			}
		}

		data, err := json.Marshal(rooms)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("encode response: %v", err)), nil
		}
		return mcp.NewToolResultText(string(data)), nil
	}
}

func makeToggleLightHandler(client *hue.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetString("id", "")
		group := req.GetString("group", "")
		name := req.GetString("name", "")

		ids, err := resolveLightIDs(ctx, client, id, group, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var toggled []string
		for _, lightID := range ids {
			if err := client.ToggleLight(ctx, lightID); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("toggle %s: %v", lightID, err)), nil
			}
			toggled = append(toggled, lightID)
		}
		return mcp.NewToolResultText(fmt.Sprintf("toggled %d light(s): %s", len(toggled), strings.Join(toggled, ", "))), nil
	}
}

func makeSetLightHandler(client *hue.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := req.GetString("id", "")
		group := req.GetString("group", "")
		name := req.GetString("name", "")

		ids, err := resolveLightIDs(ctx, client, id, group, name)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		updateReq := hue.UpdateLightRequest{}
		args := req.GetArguments()

		if _, ok := args["on"]; ok {
			v := req.GetBool("on", false)
			updateReq.On = &v
		}
		if _, ok := args["brightness"]; ok {
			v := int(req.GetFloat("brightness", 0))
			updateReq.Bri = &v
		}
		if _, ok := args["color_temp"]; ok {
			v := int(req.GetFloat("color_temp", 0))
			updateReq.CT = &v
		}

		if updateReq.On == nil && updateReq.Bri == nil && updateReq.CT == nil && updateReq.XY == nil {
			return mcp.NewToolResultError("no update fields provided (on, brightness, color_temp)"), nil
		}

		var updated []string
		for _, lightID := range ids {
			if err := client.UpdateLight(ctx, lightID, updateReq); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("update %s: %v", lightID, err)), nil
			}
			updated = append(updated, lightID)
		}
		return mcp.NewToolResultText(fmt.Sprintf("updated %d light(s): %s", len(updated), strings.Join(updated, ", "))), nil
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// resolveLightIDs returns light IDs to act on. If id is set, returns [id].
// Otherwise filters by group (required) and optionally name.
func resolveLightIDs(ctx context.Context, client *hue.Client, id, group, name string) ([]string, error) {
	if id != "" {
		return []string{id}, nil
	}
	if group == "" {
		return nil, fmt.Errorf("provide either 'id' or 'group'")
	}

	lights, err := client.ListLightsWithOptions(ctx, hue.ListLightsOptions{WithGroup: true})
	if err != nil {
		return nil, fmt.Errorf("list lights: %w", err)
	}

	var ids []string
	for _, l := range lights {
		if !strings.EqualFold(l.Group, group) {
			continue
		}
		if name != "" && !strings.EqualFold(l.Name, name) {
			continue
		}
		ids = append(ids, l.ID)
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no lights found for group %q (name filter: %q)", group, name)
	}
	return ids, nil
}
