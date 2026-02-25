package alertmanager_client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type ClientOption func(*clientOptions)

type clientOptions struct {
	username string
	password string
	headers  map[string]string
	caFile   string
	insecure bool
}

func WithBasicAuth(username, password string) ClientOption {
	return func(o *clientOptions) {
		o.username = username
		o.password = password
	}
}

func WithHeaders(headers map[string]string) ClientOption {
	return func(o *clientOptions) {
		o.headers = headers
	}
}

func WithTLSCA(caFile string) ClientOption {
	return func(o *clientOptions) {
		o.caFile = caFile
	}
}

func WithInsecure() ClientOption {
	return func(o *clientOptions) {
		o.insecure = true
	}
}

type AlertmanagerClient struct {
	baseURL    string
	httpClient *http.Client
	options    *clientOptions
}

func NewClient(baseURL string, opts ...ClientOption) (*AlertmanagerClient, error) {
	options := &clientOptions{}
	for _, opt := range opts {
		opt(options)
	}

	transport := &http.Transport{}

	if options.caFile != "" || options.insecure {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: options.insecure,
		}
		if options.caFile != "" {
			caCert, err := os.ReadFile(options.caFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA file %q: %w", options.caFile, err)
			}
			caCertPool := x509.NewCertPool()
			caCertPool.AppendCertsFromPEM(caCert)
			tlsConfig.RootCAs = caCertPool
		}
		transport.TLSClientConfig = tlsConfig
	}

	return &AlertmanagerClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Transport: transport, Timeout: 30 * time.Second},
		options:    options,
	}, nil
}

func (c *AlertmanagerClient) doRequest(method, path string, body any) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.options.username != "" {
		req.SetBasicAuth(c.options.username, c.options.password)
	}

	for k, v := range c.options.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// AlertFilter holds optional filter parameters for listing alerts.
type AlertFilter struct {
	Filter      []string
	Active      *bool
	Silenced    *bool
	Inhibited   *bool
	Unprocessed *bool
	Receiver    string
}

func (c *AlertmanagerClient) GetAlerts(filter *AlertFilter) ([]map[string]any, error) {
	path := "/api/v2/alerts"
	if filter != nil {
		params := url.Values{}
		for _, f := range filter.Filter {
			params.Add("filter", f)
		}
		if filter.Active != nil {
			params.Set("active", fmt.Sprintf("%v", *filter.Active))
		}
		if filter.Silenced != nil {
			params.Set("silenced", fmt.Sprintf("%v", *filter.Silenced))
		}
		if filter.Inhibited != nil {
			params.Set("inhibited", fmt.Sprintf("%v", *filter.Inhibited))
		}
		if filter.Unprocessed != nil {
			params.Set("unprocessed", fmt.Sprintf("%v", *filter.Unprocessed))
		}
		if filter.Receiver != "" {
			params.Set("receiver", filter.Receiver)
		}
		if encoded := params.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}

	data, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var alerts []map[string]any
	if err := json.Unmarshal(data, &alerts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alerts: %w", err)
	}
	return alerts, nil
}

// SilenceFilter holds optional filter parameters for listing silences.
type SilenceFilter struct {
	Filter []string
}

func (c *AlertmanagerClient) GetSilences(filter *SilenceFilter) ([]map[string]any, error) {
	path := "/api/v2/silences"
	if filter != nil {
		params := url.Values{}
		for _, f := range filter.Filter {
			params.Add("filter", f)
		}
		if encoded := params.Encode(); encoded != "" {
			path += "?" + encoded
		}
	}

	data, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var silences []map[string]any
	if err := json.Unmarshal(data, &silences); err != nil {
		return nil, fmt.Errorf("failed to unmarshal silences: %w", err)
	}
	return silences, nil
}

func (c *AlertmanagerClient) GetSilence(id string) (map[string]any, error) {
	data, err := c.doRequest(http.MethodGet, "/api/v2/silence/"+id, nil)
	if err != nil {
		return nil, err
	}

	var silence map[string]any
	if err := json.Unmarshal(data, &silence); err != nil {
		return nil, fmt.Errorf("failed to unmarshal silence: %w", err)
	}
	return silence, nil
}

func (c *AlertmanagerClient) CreateSilence(silence map[string]any) (string, error) {
	data, err := c.doRequest(http.MethodPost, "/api/v2/silences", silence)
	if err != nil {
		return "", err
	}

	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("failed to unmarshal create response: %w", err)
	}
	return result["silenceID"], nil
}

func (c *AlertmanagerClient) DeleteSilence(id string) error {
	_, err := c.doRequest(http.MethodDelete, "/api/v2/silence/"+id, nil)
	return err
}
