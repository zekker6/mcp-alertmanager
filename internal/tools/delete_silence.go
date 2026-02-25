package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/zekker6/mcp-alertmanager/lib/alertmanager_client"
)

func NewDeleteSilenceTool() mcp.Tool {
	return mcp.NewTool("delete_silence",
		mcp.WithDescription("Expires (deletes) a silence by its ID in Alertmanager. The silence will no longer suppress matching alerts."),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("The silence ID to expire/delete"),
		),
	)
}

func GetDeleteSilenceHandler(c *alertmanager_client.AlertmanagerClient) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, err := request.RequireString("id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		if err := c.DeleteSilence(id); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to delete silence: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Silence %s expired successfully", id)), nil
	}
}
