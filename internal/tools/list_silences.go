package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/zekker6/mcp-alertmanager/lib/alertmanager_client"
)

func NewListSilencesTool() mcp.Tool {
	return mcp.NewTool("list_silences",
		mcp.WithDescription("Lists silences from Alertmanager with optional filters. By default, expired silences are excluded."),
		mcp.WithArray("filter",
			mcp.Description("Label matchers to filter silences, e.g. ['severity=critical']. Supports =, !=, =~, !~ operators."),
			mcp.WithStringItems(),
		),
		mcp.WithBoolean("show_expired",
			mcp.Description("If true, include expired silences in the results (default: false)"),
		),
	)
}

func GetListSilencesHandler(c *alertmanager_client.AlertmanagerClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		var filter *alertmanager_client.SilenceFilter
		filterStrings := getStringArray(request, "filter")
		if len(filterStrings) > 0 {
			filter = &alertmanager_client.SilenceFilter{Filter: filterStrings}
		}

		silences, err := c.GetSilences(filter)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list silences: %v", err)), nil
		}

		// Filter out expired silences unless show_expired is true
		showExpired := getBoolPtr(request, "show_expired")
		if showExpired == nil || !*showExpired {
			filtered := make([]map[string]any, 0, len(silences))
			for _, s := range silences {
				status, ok := s["status"].(map[string]any)
				if ok {
					state, _ := status["state"].(string)
					if state == "expired" {
						continue
					}
				}
				filtered = append(filtered, s)
			}
			silences = filtered
		}

		data, err := json.MarshalIndent(silences, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal silences: %v", err)), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}
