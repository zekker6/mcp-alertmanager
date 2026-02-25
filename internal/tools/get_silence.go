package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/zekker6/mcp-alertmanager/lib/alertmanager_client"
)

func NewGetSilenceTool() mcp.Tool {
	return mcp.NewTool("get_silence",
		mcp.WithDescription("Gets a single silence by its ID from Alertmanager."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("The silence ID to retrieve"),
		),
	)
}

func GetGetSilenceHandler(c *alertmanager_client.AlertmanagerClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		silence, err := c.GetSilence(id)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get silence: %v", err)), nil
		}

		data, err := json.MarshalIndent(silence, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to marshal silence: %v", err)), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}
