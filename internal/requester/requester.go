package requester

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/oas-mcp/oas-mcp/internal/config"
	"github.com/oas-mcp/oas-mcp/internal/logger"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the requester functionality for dependency injection
var Module = fx.Options(
	fx.Provide(NewRequester),
)

// Requester handles HTTP requests to upstream APIs
type Requester struct {
	client *http.Client
	config *config.Config
}

// NewRequester creates a new requester instance
func NewRequester(cfg *config.Config) *Requester {
	client := &http.Client{
		Timeout: time.Duration(cfg.Upstream.Timeout) * time.Second,
	}

	return &Requester{
		client: client,
		config: cfg,
	}
}

// Request represents an HTTP request
type Request struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
	Query   map[string]string `json:"query,omitempty"`
	Body    interface{}       `json:"body,omitempty"`
}

// Response represents an HTTP response
type Response struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       interface{}       `json:"body"`
}

// Execute executes an HTTP request
func (r *Requester) Execute(ctx context.Context, req *Request) (*Response, error) {
	// Build the full URL
	url := r.buildURL(req.Path, req.Query)

	// Prepare request body
	var bodyReader io.Reader
	if req.Body != nil {
		bodyData, err := json.Marshal(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyData)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	r.setHeaders(httpReq, req.Headers)

	// Set authentication
	r.setAuthentication(httpReq)

	// Log request
	logger.Debug("Executing HTTP request",
		zap.String("method", req.Method),
		zap.String("url", url),
		zap.Any("headers", req.Headers),
		zap.Any("query", req.Query))

	// Execute request
	httpResp, err := r.client.Do(httpReq)
	if err != nil {
		logger.Error("HTTP request failed",
			zap.String("method", req.Method),
			zap.String("url", url),
			zap.Error(err))
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response body
	var body interface{}
	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, &body); err != nil {
			// If JSON parsing fails, return as string
			body = string(respBody)
		}
	}

	// Extract response headers
	headers := make(map[string]string)
	for key, values := range httpResp.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}

	response := &Response{
		StatusCode: httpResp.StatusCode,
		Headers:    headers,
		Body:       body,
	}

	// Log response
	logger.Debug("HTTP response received",
		zap.String("method", req.Method),
		zap.String("url", url),
		zap.Int("status_code", response.StatusCode),
		zap.Any("headers", response.Headers))

	return response, nil
}

// buildURL builds the full URL from base URL, path, and query parameters
func (r *Requester) buildURL(path string, query map[string]string) string {
	url := r.config.Upstream.BaseURL
	if url == "" {
		url = "http://localhost"
	}

	// Ensure URL ends with path
	if path != "" {
		if url[len(url)-1] != '/' && path[0] != '/' {
			url += "/"
		}
		url += path
	}

	// Add query parameters
	if len(query) > 0 {
		url += "?"
		first := true
		for key, value := range query {
			if !first {
				url += "&"
			}
			url += fmt.Sprintf("%s=%s", key, value)
			first = false
		}
	}

	return url
}

// setHeaders sets request headers
func (r *Requester) setHeaders(req *http.Request, headers map[string]string) {
	// Set default headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "oas-mcp/1.0")

	// Set Content-Type for requests with body
	if req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}
}

// setAuthentication sets authentication headers based on configuration
func (r *Requester) setAuthentication(req *http.Request) {
	switch r.config.Auth.Type {
	case "bearer":
		if r.config.Auth.Token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.config.Auth.Token))
		}
	case "basic":
		if r.config.Auth.Username != "" && r.config.Auth.Password != "" {
			req.SetBasicAuth(r.config.Auth.Username, r.config.Auth.Password)
		}
	case "apikey":
		if r.config.Auth.APIKey != "" {
			// Assume API key goes in Authorization header, but this could be configurable
			req.Header.Set("Authorization", fmt.Sprintf("ApiKey %s", r.config.Auth.APIKey))
		}
		if r.config.Auth.Token != "" {
			// Alternative: use X-API-Key header
			req.Header.Set("X-API-Key", r.config.Auth.Token)
		}
	case "none":
		// No authentication
	default:
		logger.Warn("Unknown authentication type", zap.String("type", r.config.Auth.Type))
	}
}
