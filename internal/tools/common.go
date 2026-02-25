package tools

import (
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// parseMatcher parses a matcher string like "name=value", "name!=value",
// "name=~regex", or "name!~regex" into the Alertmanager v2 API matcher format.
func parseMatcher(s string) (map[string]any, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty matcher")
	}

	// Try operators in order: !~, =~, !=, =
	for _, op := range []string{"!~", "=~", "!=", "="} {
		idx := strings.Index(s, op)
		if idx < 0 {
			continue
		}
		name := strings.TrimSpace(s[:idx])
		value := strings.TrimSpace(s[idx+len(op):])
		if name == "" {
			return nil, fmt.Errorf("matcher has empty name: %q", s)
		}
		return map[string]any{
			"name":    name,
			"value":   value,
			"isRegex": op == "=~" || op == "!~",
			"isEqual": op == "=" || op == "=~",
		}, nil
	}

	return nil, fmt.Errorf("invalid matcher format: %q (expected name=value, name!=value, name=~regex, or name!~regex)", s)
}

// parseMatchers parses a slice of matcher strings into Alertmanager v2 API format.
func parseMatchers(matchers []string) ([]map[string]any, error) {
	result := make([]map[string]any, 0, len(matchers))
	for _, m := range matchers {
		parsed, err := parseMatcher(m)
		if err != nil {
			return nil, err
		}
		result = append(result, parsed)
	}
	return result, nil
}

// computeSilenceTimes computes startsAt and endsAt from the tool parameters.
// If startsAt/endsAt are provided, they are used directly. Otherwise, duration is used from now.
func computeSilenceTimes(startsAt, endsAt, duration string) (string, string, error) {
	if startsAt != "" && endsAt != "" {
		// Validate RFC3339
		if _, err := time.Parse(time.RFC3339, startsAt); err != nil {
			return "", "", fmt.Errorf("invalid starts_at: %w", err)
		}
		if _, err := time.Parse(time.RFC3339, endsAt); err != nil {
			return "", "", fmt.Errorf("invalid ends_at: %w", err)
		}
		return startsAt, endsAt, nil
	}

	if duration == "" {
		duration = "1h"
	}
	d, err := time.ParseDuration(duration)
	if err != nil {
		return "", "", fmt.Errorf("invalid duration: %w", err)
	}

	now := time.Now().UTC()
	if startsAt != "" {
		start, err := time.Parse(time.RFC3339, startsAt)
		if err != nil {
			return "", "", fmt.Errorf("invalid starts_at: %w", err)
		}
		return startsAt, start.Add(d).Format(time.RFC3339), nil
	}

	if endsAt != "" {
		if _, err := time.Parse(time.RFC3339, endsAt); err != nil {
			return "", "", fmt.Errorf("invalid ends_at: %w", err)
		}
		return now.Format(time.RFC3339), endsAt, nil
	}

	return now.Format(time.RFC3339), now.Add(d).Format(time.RFC3339), nil
}

// getStringArray extracts a string array parameter from the MCP request arguments.
func getStringArray(request mcp.CallToolRequest, key string) []string {
	args := request.GetArguments()
	val, ok := args[key]
	if !ok || val == nil {
		return nil
	}

	switch v := val.(type) {
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return v
	case string:
		// Single string treated as single-element array
		if v != "" {
			return []string{v}
		}
		return nil
	default:
		return nil
	}
}

// getBoolPtr extracts an optional boolean parameter, returning nil if not set.
func getBoolPtr(request mcp.CallToolRequest, key string) *bool {
	args := request.GetArguments()
	val, ok := args[key]
	if !ok || val == nil {
		return nil
	}
	if b, ok := val.(bool); ok {
		return &b
	}
	return nil
}
