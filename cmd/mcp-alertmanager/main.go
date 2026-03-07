package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/zap"

	"github.com/zekker6/mcp-alertmanager/internal/tools"
	"github.com/zekker6/mcp-alertmanager/lib/alertmanager_client"
	"github.com/zekker6/mcp-alertmanager/lib/logger"
)

func readPasswordFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read password file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

type headerFlags []string

func (h *headerFlags) String() string { return strings.Join(*h, ", ") }
func (h *headerFlags) Set(value string) error {
	*h = append(*h, value)
	return nil
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	alertmanagerURL = flag.String("url", "", "Alertmanager base URL (e.g. http://alertmanager:9093)")

	mode                 = flag.String("mode", "stdio", "Mode to run the MCP server in (stdio, sse, http)")
	httpListenAddr       = flag.String("httpListenAddr", ":8012", "Address to listen for http connections")
	heartbeatInterval    = flag.Duration("httpHeartbeatInterval", 30*time.Second, "Interval for sending heartbeat messages (http mode)")
	sseKeepAliveInterval = flag.Duration("sseKeepAliveInterval", 30*time.Second, "Interval for sending keep-alive messages (sse mode)")

	username     = flag.String("username", "", "Username for basic authentication (requires -password-file)")
	passwordFile = flag.String("password-file", "", "Path to file containing password for basic authentication (requires -username)")

	tlsCAFile             = flag.String("tls-ca", "", "Path to CA certificate file for TLS verification")
	tlsInsecureSkipVerify = flag.Bool("tls-insecure-skip-verify", false, "Skip TLS certificate verification (insecure)")
)

var headers headerFlags

func init() {
	flag.Var(&headers, "header", "Custom HTTP header as 'Key: Value' (repeatable)")
}

func main() {
	flag.Parse()

	logger.Init()
	defer logger.Stop()

	if *alertmanagerURL == "" {
		logger.Error("Alertmanager URL is required. Use -url to set it")
		os.Exit(1)
	}

	switch *mode {
	case "stdio", "sse", "http":
	default:
		logger.Error("Invalid mode specified. Supported modes are 'stdio', 'sse', and 'http'", zap.String("mode", *mode))
		os.Exit(1)
	}

	if (*mode == "sse" || *mode == "http") && *httpListenAddr == "" {
		logger.Error("HTTP listen address must be specified. Use -httpListenAddr to set it", zap.String("mode", *mode))
		os.Exit(1)
	}

	amClient, err := getAlertmanagerClient()
	if err != nil {
		logger.Error("Failed to create Alertmanager client", zap.Error(err))
		os.Exit(1)
	}

	if err := amClient.HealthCheck(); err != nil {
		logger.Error("Alertmanager is not reachable", zap.String("url", *alertmanagerURL), zap.Error(err))
		os.Exit(1)
	}
	logger.Info("Alertmanager health check passed", zap.String("url", *alertmanagerURL))

	s := server.NewMCPServer(
		"Alertmanager MCP Server",
		fmt.Sprintf("v%s (commit: %s, date: %s)", version, commit, date),
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	s.AddTool(tools.NewListAlertsTool(), tools.GetListAlertsHandler(amClient))
	s.AddTool(tools.NewListSilencesTool(), tools.GetListSilencesHandler(amClient))
	s.AddTool(tools.NewGetSilenceTool(), tools.GetGetSilenceHandler(amClient))
	s.AddTool(tools.NewCreateSilenceTool(), tools.GetCreateSilenceHandler(amClient))
	s.AddTool(tools.NewDeleteSilenceTool(), tools.GetDeleteSilenceHandler(amClient))

	logger.Info("Starting Alertmanager MCP server",
		zap.String("version", version),
		zap.String("commit", commit),
		zap.String("date", date),
		zap.String("mode", *mode),
		zap.String("url", *alertmanagerURL),
	)

	switch *mode {
	case "stdio":
		if err := server.ServeStdio(s); err != nil {
			logger.Error("Failed to start MCP server in stdio mode", zap.Error(err))
		}
	case "sse":
		var opts []server.SSEOption
		if *sseKeepAliveInterval > 0 {
			opts = append(opts, server.WithKeepAliveInterval(*sseKeepAliveInterval))
		}
		srv := server.NewSSEServer(s, opts...)
		if err := srv.Start(*httpListenAddr); err != nil {
			logger.Error("Failed to start SSE server", zap.Error(err))
		}
	case "http":
		var opts []server.StreamableHTTPOption
		if *heartbeatInterval > 0 {
			opts = append(opts, server.WithHeartbeatInterval(*heartbeatInterval))
		}
		srv := server.NewStreamableHTTPServer(s, opts...)
		if err := srv.Start(*httpListenAddr); err != nil {
			logger.Error("Failed to start HTTP server", zap.Error(err))
		}
	}
}

func getAlertmanagerClient() (*alertmanager_client.AlertmanagerClient, error) {
	var clientOpts []alertmanager_client.ClientOption

	// Validate basic auth flags
	if (*username != "") != (*passwordFile != "") {
		if *username != "" {
			logger.Error("Both -username and -password-file must be provided together (missing -password-file)")
		} else {
			logger.Error("Both -username and -password-file must be provided together (missing -username)")
		}
		os.Exit(1)
	}

	if *username != "" && *passwordFile != "" {
		password, err := readPasswordFile(*passwordFile)
		if err != nil {
			logger.Error("Failed to read password file", zap.Error(err))
			os.Exit(1)
		}
		clientOpts = append(clientOpts, alertmanager_client.WithBasicAuth(*username, password))
	}

	if len(headers) > 0 {
		headerMap := make(map[string]string)
		for _, h := range headers {
			parts := strings.SplitN(h, ":", 2)
			if len(parts) != 2 {
				logger.Error("Invalid header format, expected 'Key: Value'", zap.String("header", h))
				os.Exit(1)
			}
			key := strings.TrimSpace(parts[0])
			if key == "" {
				logger.Error("Invalid header format, header name is empty", zap.String("header", h))
				os.Exit(1)
			}
			headerMap[key] = strings.TrimSpace(parts[1])
		}
		clientOpts = append(clientOpts, alertmanager_client.WithHeaders(headerMap))
	}

	if *tlsCAFile != "" {
		clientOpts = append(clientOpts, alertmanager_client.WithTLSCA(*tlsCAFile))
	}
	if *tlsInsecureSkipVerify {
		clientOpts = append(clientOpts, alertmanager_client.WithInsecure())
	}

	return alertmanager_client.NewClient(*alertmanagerURL, clientOpts...)
}
