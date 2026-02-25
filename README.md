# mcp-alertmanager

MCP server for Prometheus Alertmanager. Exposes alert listing and silence management as MCP tools.

## Tools

| Tool | Description |
|------|-------------|
| `list_alerts` | List alerts with optional filters (label matchers, state, receiver) |
| `list_silences` | List silences with optional label matcher filters |
| `get_silence` | Get a single silence by ID |
| `create_silence` | Create a new silence with matchers, author, comment, and duration |
| `delete_silence` | Expire (delete) a silence by ID |

## Usage

### stdio mode (default)

```bash
mcp-alertmanager -url http://alertmanager:9093
```

### SSE mode

```bash
mcp-alertmanager -url http://alertmanager:9093 -mode sse -httpListenAddr :8012
```

### Authentication

**Basic auth:**

```bash
mcp-alertmanager -url http://alertmanager:9093 -username admin -password-file /path/to/password
```

**Custom headers (e.g. bearer token, multi-tenancy):**

```bash
mcp-alertmanager -url http://alertmanager:9093 \
  -header "Authorization: Bearer <token>" \
  -header "X-Scope-OrgID: tenant1"
```

### Claude Desktop Configuration

```json
{
  "mcpServers": {
    "alertmanager": {
      "command": "mcp-alertmanager",
      "args": ["-url", "http://alertmanager:9093"]
    }
  }
}
```

## Building

```bash
task build
```

## Testing

```bash
task test        # unit tests
task test:e2e    # e2e tests (builds binary first)
task test:all    # all tests
```
