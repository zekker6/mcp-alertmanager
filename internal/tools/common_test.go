package tools

import (
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestParseMatcher(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantName  string
		wantValue string
		wantRegex bool
		wantEqual bool
		wantErr   bool
	}{
		{
			name: "exact match", input: "severity=critical",
			wantName: "severity", wantValue: "critical", wantRegex: false, wantEqual: true,
		},
		{
			name: "negative match", input: "env!=staging",
			wantName: "env", wantValue: "staging", wantRegex: false, wantEqual: false,
		},
		{
			name: "regex match", input: "instance=~node.*",
			wantName: "instance", wantValue: "node.*", wantRegex: true, wantEqual: true,
		},
		{
			name: "negative regex match", input: "job!~test.*",
			wantName: "job", wantValue: "test.*", wantRegex: true, wantEqual: false,
		},
		{
			name: "whitespace trimmed", input: "  alertname = TestAlert  ",
			wantName: "alertname", wantValue: "TestAlert", wantRegex: false, wantEqual: true,
		},
		{
			name: "empty string", input: "",
			wantErr: true,
		},
		{
			name: "no operator", input: "justlabel",
			wantErr: true,
		},
		{
			name: "empty name", input: "=value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseMatcher(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result["name"] != tt.wantName {
				t.Errorf("name = %q, want %q", result["name"], tt.wantName)
			}
			if result["value"] != tt.wantValue {
				t.Errorf("value = %q, want %q", result["value"], tt.wantValue)
			}
			if result["isRegex"] != tt.wantRegex {
				t.Errorf("isRegex = %v, want %v", result["isRegex"], tt.wantRegex)
			}
			if result["isEqual"] != tt.wantEqual {
				t.Errorf("isEqual = %v, want %v", result["isEqual"], tt.wantEqual)
			}
		})
	}
}

func TestComputeSilenceTimes(t *testing.T) {
	tests := []struct {
		name     string
		startsAt string
		endsAt   string
		duration string
		wantErr  bool
	}{
		{
			name:     "explicit times",
			startsAt: "2026-01-01T00:00:00Z",
			endsAt:   "2026-01-02T00:00:00Z",
			duration: "",
			wantErr:  false,
		},
		{
			name:     "duration only",
			startsAt: "",
			endsAt:   "",
			duration: "2h",
			wantErr:  false,
		},
		{
			name:     "default duration",
			startsAt: "",
			endsAt:   "",
			duration: "",
			wantErr:  false,
		},
		{
			name:     "ends_at only",
			startsAt: "",
			endsAt:   "2026-01-01T02:00:00Z",
			duration: "",
			wantErr:  false,
		},
		{
			name:     "invalid ends_at only",
			startsAt: "",
			endsAt:   "not-a-date",
			duration: "",
			wantErr:  true,
		},
		{
			name:     "invalid starts_at",
			startsAt: "not-a-date",
			endsAt:   "2026-01-02T00:00:00Z",
			duration: "",
			wantErr:  true,
		},
		{
			name:     "invalid duration",
			startsAt: "",
			endsAt:   "",
			duration: "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := computeSilenceTimes(tt.startsAt, tt.endsAt, tt.duration)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Validate RFC3339 format
			if _, err := time.Parse(time.RFC3339, start); err != nil {
				t.Errorf("startsAt is not valid RFC3339: %q", start)
			}
			if _, err := time.Parse(time.RFC3339, end); err != nil {
				t.Errorf("endsAt is not valid RFC3339: %q", end)
			}
		})
	}
}

func TestGetStringArray(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]any
		key       string
		want      []string
	}{
		{
			name:      "array of any",
			arguments: map[string]any{"filter": []any{"a=b", "c=d"}},
			key:       "filter",
			want:      []string{"a=b", "c=d"},
		},
		{
			name:      "single string",
			arguments: map[string]any{"filter": "a=b"},
			key:       "filter",
			want:      []string{"a=b"},
		},
		{
			name:      "missing key",
			arguments: map[string]any{},
			key:       "filter",
			want:      nil,
		},
		{
			name:      "nil value",
			arguments: map[string]any{"filter": nil},
			key:       "filter",
			want:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "test",
					Arguments: tt.arguments,
				},
			}
			got := getStringArray(request, tt.key)
			if len(got) != len(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestGetBoolPtr(t *testing.T) {
	tests := []struct {
		name      string
		arguments map[string]any
		key       string
		wantNil   bool
		wantVal   bool
	}{
		{
			name:      "true value",
			arguments: map[string]any{"active": true},
			key:       "active",
			wantNil:   false,
			wantVal:   true,
		},
		{
			name:      "false value",
			arguments: map[string]any{"active": false},
			key:       "active",
			wantNil:   false,
			wantVal:   false,
		},
		{
			name:      "missing key",
			arguments: map[string]any{},
			key:       "active",
			wantNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      "test",
					Arguments: tt.arguments,
				},
			}
			got := getBoolPtr(request, tt.key)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %v", *got)
				}
				return
			}
			if got == nil {
				t.Errorf("expected %v, got nil", tt.wantVal)
				return
			}
			if *got != tt.wantVal {
				t.Errorf("got %v, want %v", *got, tt.wantVal)
			}
		})
	}
}
