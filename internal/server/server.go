package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/feitianbubu/oas-mcp/internal/config"
	"github.com/feitianbubu/oas-mcp/internal/logger"
	"github.com/feitianbubu/oas-mcp/internal/parser"
	"github.com/feitianbubu/oas-mcp/internal/requester"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// Module provides the server functionality for dependency injection
var Module = fx.Options(
	fx.Provide(NewServer),
)

// Server represents the MCP server
type Server struct {
	config    *config.Config
	parser    *parser.Parser
	requester *requester.Requester
	spec      *parser.OpenAPISpec
	tools     []Tool
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config, p *parser.Parser, r *requester.Requester) (*Server, error) {
	logger.Info("Parsing OpenAPI specification", zap.String("source", cfg.SwaggerFile))

	// Parse the OpenAPI specification
	spec, err := p.ParseFile(cfg.SwaggerFile)
	if err != nil {
		logger.Error("Failed to parse OpenAPI specification",
			zap.String("source", cfg.SwaggerFile),
			zap.Error(err))
		return nil, fmt.Errorf("failed to parse OpenAPI spec from %s: %w", cfg.SwaggerFile, err)
	}

	logger.Info("Successfully parsed OpenAPI specification",
		zap.String("title", spec.Info.Title),
		zap.String("version", spec.Info.Version))

	server := &Server{
		config:    cfg,
		parser:    p,
		requester: r,
		spec:      spec,
	}

	// Generate tools from the OpenAPI spec
	server.generateTools()

	return server, nil
}

// Tool represents an MCP tool
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema Schema `json:"inputSchema"`
	Operation   *parser.OperationInfo
}

// Schema represents a JSON schema for tool input
type Schema struct {
	Type       string              `json:"type"`
	Properties map[string]Property `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

// Property represents a property in a JSON schema
type Property struct {
	Type        string      `json:"type"`
	Description string      `json:"description,omitempty"`
	Items       *Property   `json:"items,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Example     interface{} `json:"example,omitempty"`
}

// MCPRequest represents an MCP request
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse represents an MCP response
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Start starts the MCP server
func (s *Server) Start(ctx context.Context) error {
	logger.Info("Starting MCP server",
		zap.String("mode", s.config.Server.Mode),
		zap.String("swagger_file", s.config.SwaggerFile),
		zap.Int("tools_count", len(s.tools)))

	switch s.config.Server.Mode {
	case config.ServerModeSTDIO:
		return s.startSTDIOServer(ctx)
	case config.ServerModeHTTP:
		return s.startHTTPServer(ctx)
	case config.ServerModeSSE:
		return s.startSSEServer(ctx)
	default:
		return fmt.Errorf("unsupported server mode: %s", s.config.Server.Mode)
	}
}

// startSTDIOServer starts the STDIO server
func (s *Server) startSTDIOServer(ctx context.Context) error {
	logger.Info("Starting STDIO MCP server")

	// Read from stdin and write to stdout
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			var request MCPRequest
			if err := decoder.Decode(&request); err != nil {
				if err == io.EOF {
					return nil
				}
				logger.Error("Failed to decode request", zap.Error(err))
				continue
			}

			response := s.handleRequest(&request)
			if err := encoder.Encode(response); err != nil {
				logger.Error("Failed to encode response", zap.Error(err))
			}
		}
	}
}

// startHTTPServer starts the HTTP server
func (s *Server) startHTTPServer(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	logger.Info("Starting HTTP MCP server", zap.String("address", addr))

	mux := http.NewServeMux()

	// Setup static file server for public directory
	publicDir := "public"
	if _, err := os.Stat(publicDir); err == nil {
		fileServer := http.FileServer(http.Dir(publicDir))
		mux.Handle("/", fileServer)
		logger.Info("Static file server enabled", zap.String("directory", publicDir))
	} else {
		logger.Warn("Public directory not found, static file serving disabled", zap.String("directory", publicDir))
	}

	// Handle MCP requests on /mcp endpoint
	mux.HandleFunc("/mcp", s.handleHTTPRequest)

	// Handle config API requests
	mux.HandleFunc("/api/config", s.handleConfigRequest)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}

// startSSEServer starts the SSE server
func (s *Server) startSSEServer(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	logger.Info("Starting SSE MCP server", zap.String("address", addr))

	mux := http.NewServeMux()

	// Setup static file server for public directory
	publicDir := "public"
	if _, err := os.Stat(publicDir); err == nil {
		fileServer := http.FileServer(http.Dir(publicDir))
		mux.Handle("/", fileServer)
		logger.Info("Static file server enabled", zap.String("directory", publicDir))
	} else {
		logger.Warn("Public directory not found, static file serving disabled", zap.String("directory", publicDir))
	}

	mux.HandleFunc("/sse", s.handleSSERequest)

	// Handle config API requests
	mux.HandleFunc("/api/config", s.handleConfigRequest)

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}

// handleHTTPRequest handles HTTP requests
func (s *Server) handleHTTPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := s.handleRequest(&request)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSSERequest handles SSE requests
func (s *Server) handleSSERequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// This is a simplified SSE implementation
	// In a real implementation, you'd handle SSE protocol properly
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	for {
		select {
		case <-r.Context().Done():
			return
		default:
			// Handle SSE messages here
			flusher.Flush()
		}
	}
}

// handleRequest handles MCP requests
func (s *Server) handleRequest(request *MCPRequest) *MCPResponse {
	logger.Debug("Handling MCP request",
		zap.String("method", request.Method),
		zap.Any("id", request.ID))

	switch request.Method {
	case "initialize":
		return s.handleInitialize(request)
	case "tools/list":
		return s.handleToolsList(request)
	case "tools/call":
		return s.handleToolsCall(request)
	default:
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: "Method not found",
			},
		}
	}
}

// handleInitialize handles initialize requests
func (s *Server) handleInitialize(request *MCPRequest) *MCPResponse {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "oas-mcp",
			"version": "1.0.0",
		},
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  result,
	}
}

// handleToolsList handles tools/list requests
func (s *Server) handleToolsList(request *MCPRequest) *MCPResponse {
	result := map[string]interface{}{
		"tools": s.tools,
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  result,
	}
}

// handleToolsCall handles tools/call requests
func (s *Server) handleToolsCall(request *MCPRequest) *MCPResponse {
	params, ok := request.Params.(map[string]interface{})
	if !ok {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
	}

	toolName, ok := params["name"].(string)
	if !ok {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Missing tool name",
			},
		}
	}

	arguments, _ := params["arguments"].(map[string]interface{})

	// Find the tool
	var tool *Tool
	for _, t := range s.tools {
		if t.Name == toolName {
			tool = &t
			break
		}
	}

	if tool == nil {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Tool not found",
			},
		}
	}

	// Execute the tool
	result, err := s.executeTool(tool, arguments)
	if err != nil {
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &MCPError{
				Code:    -32603,
				Message: err.Error(),
			},
		}
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": result,
				},
			},
		},
	}
}

// executeTool executes a tool
func (s *Server) executeTool(tool *Tool, arguments map[string]interface{}) (string, error) {
	logger.Debug("Executing tool",
		zap.String("tool", tool.Name),
		zap.Any("arguments", arguments))

	// Build request from arguments
	req := &requester.Request{
		Method:  tool.Operation.Method,
		Path:    tool.Operation.Path,
		Headers: make(map[string]string),
		Query:   make(map[string]string),
	}

	// Extract parameters from arguments based on OpenAPI spec
	for _, param := range tool.Operation.Operation.Parameters {
		if value, exists := arguments[param.Name]; exists {
			switch param.In {
			case "query":
				req.Query[param.Name] = fmt.Sprintf("%v", value)
			case "header":
				req.Headers[param.Name] = fmt.Sprintf("%v", value)
			case "path":
				// Replace path parameters
				req.Path = strings.ReplaceAll(req.Path, "{"+param.Name+"}", fmt.Sprintf("%v", value))
			}
		}
	}

	// Handle request body
	if tool.Operation.Operation.RequestBody != nil {
		if bodyData, exists := arguments["body"]; exists {
			req.Body = bodyData
		}
	}

	// Execute the request
	ctx := context.Background()
	response, err := s.requester.Execute(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}

	// Format response
	resultData, _ := json.MarshalIndent(map[string]interface{}{
		"status_code": response.StatusCode,
		"headers":     response.Headers,
		"body":        response.Body,
	}, "", "  ")

	return string(resultData), nil
}

// generateTools generates MCP tools from OpenAPI operations
func (s *Server) generateTools() {
	operations := s.parser.GetOperations(s.spec)

	for _, op := range operations {
		tool := Tool{
			Name:        s.generateToolName(op),
			Description: s.generateToolDescription(op),
			InputSchema: s.generateInputSchema(op),
			Operation:   &op,
		}

		s.tools = append(s.tools, tool)
	}

	logger.Info("Generated tools from OpenAPI spec", zap.Int("count", len(s.tools)))
}

// generateToolName generates a tool name from an operation
func (s *Server) generateToolName(op parser.OperationInfo) string {
	if op.OperationID != "" {
		return op.OperationID
	}
	return fmt.Sprintf("%s_%s", strings.ToLower(op.Method), strings.ReplaceAll(strings.Trim(op.Path, "/"), "/", "_"))
}

// generateToolDescription generates a tool description from an operation
func (s *Server) generateToolDescription(op parser.OperationInfo) string {
	if op.Operation.Description != "" {
		return op.Operation.Description
	}
	if op.Operation.Summary != "" {
		return op.Operation.Summary
	}
	return fmt.Sprintf("%s %s", op.Method, op.Path)
}

// generateInputSchema generates an input schema from an operation
func (s *Server) generateInputSchema(op parser.OperationInfo) Schema {
	schema := Schema{
		Type:       "object",
		Properties: make(map[string]Property),
		Required:   []string{},
	}

	// Add parameters
	for _, param := range op.Operation.Parameters {
		property := Property{
			Type:        "string", // Simplified - should be based on param.Schema.Type
			Description: param.Description,
		}

		if param.Schema != nil && param.Schema.Type != "" {
			property.Type = param.Schema.Type
		}

		schema.Properties[param.Name] = property

		if param.Required {
			schema.Required = append(schema.Required, param.Name)
		}
	}

	// Add request body
	if op.Operation.RequestBody != nil {
		schema.Properties["body"] = Property{
			Type:        "object",
			Description: op.Operation.RequestBody.Description,
		}

		if op.Operation.RequestBody.Required {
			schema.Required = append(schema.Required, "body")
		}
	}

	return schema
}

// handleConfigRequest handles config API requests
func (s *Server) handleConfigRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"swagger_file": s.config.SwaggerFile,
	})
}
