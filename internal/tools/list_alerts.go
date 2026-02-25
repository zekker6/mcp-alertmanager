package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/zekker6/mcp-alertmanager/lib/alertmanager_client"
)

func NewListAlertsTool() mcp.Tool {
	return mcp.NewTool("list_alerts",
		mcp.WithDescription("Lists alerts from Alertmanager with optional filters. Returns alerts matching the specified criteria."),
		mcp.WithArray("filter",
			mcp.Description("Label matchers to filter alerts, e.g. ['severity=critical']. Supports =, !=, =~, !~ operators."),
			mcp.WithStringItems(),
		),
		mcp.WithBoolean("active",
			mcp.Description("If true, only show active (firing) alerts"),
		),
		mcp.WithBoolean("silenced",
			mcp.Description("If true, only show silenced alerts"),
		),
		mcp.WithBoolean("inhibited",
			mcp.Description("If true, only show inhibited alerts"),
		),
		mcp.WithBoolean("unprocessed",
			mcp.Description("If true, only show unprocessed alerts"),
		),
		mcp.WithString("receiver",
			mcp.Description("Filter alerts by receiver name"),
		),
	)
}

func GetListAlertsHandler(c *alertmanager_client.AlertmanagerClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		filter := &alertmanager_client.AlertFilter{
			Filter:      getStringArray(request, "filter"),
			Active:      getBoolPtr(request, "active"),
			Silenced:    getBoolPtr(request, "silenced"),
			Inhibited:   getBoolPtr(request, "inhibited"),
			Unprocessed: getBoolPtr(request, "unprocessed"),
			Receiver:    request.GetString("receiver", ""),
		}

		alerts, err := c.GetAlerts(filter)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list alerts: %v", err)), nil
		}

		data, err := json.MarshalIndent(alerts, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal alerts: %v", err)), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}
