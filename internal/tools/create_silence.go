package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/zekker6/mcp-alertmanager/lib/alertmanager_client"
)

func NewCreateSilenceTool() mcp.Tool {
	return mcp.NewTool("create_silence",
		mcp.WithDescription("Creates a new silence in Alertmanager. Silences suppress alerts matching the specified matchers for the given duration."),
		mcp.WithArray("matchers",
			mcp.Required(),
			mcp.Description("Label matchers for the silence. Format: ['label=value', 'label!=value', 'label=~regex', 'label!~regex']."),
			mcp.WithStringItems(),
		),
		mcp.WithString("author",
			mcp.Required(),
			mcp.Description("Name of the person creating the silence"),
		),
		mcp.WithString("comment",
			mcp.Required(),
			mcp.Description("Reason for creating the silence"),
		),
		mcp.WithString("duration",
			mcp.Description("Duration of the silence as a Go duration string (e.g. '1h', '30m', '2h30m'). Default: '1h'. Ignored if both starts_at and ends_at are provided."),
		),
		mcp.WithString("starts_at",
			mcp.Description("Start time in RFC3339 format (e.g. '2026-01-01T00:00:00Z'). If omitted, starts now."),
		),
		mcp.WithString("ends_at",
			mcp.Description("End time in RFC3339 format (e.g. '2026-01-01T01:00:00Z'). If omitted, computed from starts_at + duration."),
		),
	)
}

func GetCreateSilenceHandler(c *alertmanager_client.AlertmanagerClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		matcherStrings := getStringArray(request, "matchers")
		if len(matcherStrings) == 0 {
			return mcp.NewToolResultError("matchers is required and must not be empty"), nil
		}

		author, err := request.RequireString("author")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		comment, err := request.RequireString("comment")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		duration := request.GetString("duration", "1h")
		startsAt := request.GetString("starts_at", "")
		endsAt := request.GetString("ends_at", "")

		matchers, err := parseMatchers(matcherStrings)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid matcher: %v", err)), nil
		}

		computedStart, computedEnd, err := computeSilenceTimes(startsAt, endsAt, duration)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("invalid time parameters: %v", err)), nil
		}

		silence := map[string]any{
			"matchers":  matchers,
			"createdBy": author,
			"comment":   comment,
			"startsAt":  computedStart,
			"endsAt":    computedEnd,
		}

		silenceID, err := c.CreateSilence(silence)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create silence: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Silence created successfully. ID: %s", silenceID)), nil
	}
}
