package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"go.uber.org/fx"
	"gopkg.in/yaml.v3"
)

// Module provides the parser functionality for dependency injection
var Module = fx.Options(
	fx.Provide(NewParser),
)

// Parser handles OpenAPI/Swagger document parsing
type Parser struct{}

// NewParser creates a new parser instance
func NewParser() *Parser {
	return &Parser{}
}

// OpenAPISpec represents the basic structure of an OpenAPI specification
type OpenAPISpec struct {
	OpenAPI    string                `json:"openapi" yaml:"openapi"`
	Swagger    string                `json:"swagger" yaml:"swagger"`
	Info       Info                  `json:"info" yaml:"info"`
	Servers    []Server              `json:"servers,omitempty" yaml:"servers,omitempty"`
	Host       string                `json:"host,omitempty" yaml:"host,omitempty"`
	BasePath   string                `json:"basePath,omitempty" yaml:"basePath,omitempty"`
	Schemes    []string              `json:"schemes,omitempty" yaml:"schemes,omitempty"`
	Paths      map[string]PathItem   `json:"paths" yaml:"paths"`
	Components *Components           `json:"components,omitempty" yaml:"components,omitempty"`
	Security   []SecurityRequirement `json:"security,omitempty" yaml:"security,omitempty"`
	Tags       []Tag                 `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// Info represents the info section of an OpenAPI spec
type Info struct {
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Version     string `json:"version" yaml:"version"`
}

// Server represents a server in OpenAPI 3.x
type Server struct {
	URL         string                    `json:"url" yaml:"url"`
	Description string                    `json:"description,omitempty" yaml:"description,omitempty"`
	Variables   map[string]ServerVariable `json:"variables,omitempty" yaml:"variables,omitempty"`
}

// ServerVariable represents a server variable
type ServerVariable struct {
	Enum        []string `json:"enum,omitempty" yaml:"enum,omitempty"`
	Default     string   `json:"default" yaml:"default"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
}

// PathItem represents a path in the OpenAPI spec
type PathItem struct {
	Get     *Operation `json:"get,omitempty" yaml:"get,omitempty"`
	Post    *Operation `json:"post,omitempty" yaml:"post,omitempty"`
	Put     *Operation `json:"put,omitempty" yaml:"put,omitempty"`
	Delete  *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
	Options *Operation `json:"options,omitempty" yaml:"options,omitempty"`
	Head    *Operation `json:"head,omitempty" yaml:"head,omitempty"`
	Patch   *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
	Trace   *Operation `json:"trace,omitempty" yaml:"trace,omitempty"`
}

// Operation represents an HTTP operation
type Operation struct {
	OperationID string                `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Summary     string                `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string                `json:"description,omitempty" yaml:"description,omitempty"`
	Tags        []string              `json:"tags,omitempty" yaml:"tags,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses" yaml:"responses"`
	Security    []SecurityRequirement `json:"security,omitempty" yaml:"security,omitempty"`
}

// Parameter represents a parameter in the OpenAPI spec
type Parameter struct {
	Name        string      `json:"name" yaml:"name"`
	In          string      `json:"in" yaml:"in"`
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Required    bool        `json:"required,omitempty" yaml:"required,omitempty"`
	Schema      *Schema     `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example     interface{} `json:"example,omitempty" yaml:"example,omitempty"`
}

// RequestBody represents a request body
type RequestBody struct {
	Description string               `json:"description,omitempty" yaml:"description,omitempty"`
	Content     map[string]MediaType `json:"content" yaml:"content"`
	Required    bool                 `json:"required,omitempty" yaml:"required,omitempty"`
}

// Response represents a response
type Response struct {
	Description string               `json:"description" yaml:"description"`
	Content     map[string]MediaType `json:"content,omitempty" yaml:"content,omitempty"`
	Headers     map[string]Header    `json:"headers,omitempty" yaml:"headers,omitempty"`
}

// MediaType represents a media type
type MediaType struct {
	Schema   *Schema     `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example  interface{} `json:"example,omitempty" yaml:"example,omitempty"`
	Examples interface{} `json:"examples,omitempty" yaml:"examples,omitempty"`
}

// Header represents a header
type Header struct {
	Description string      `json:"description,omitempty" yaml:"description,omitempty"`
	Schema      *Schema     `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example     interface{} `json:"example,omitempty" yaml:"example,omitempty"`
}

// Schema represents a JSON schema
type Schema struct {
	Type        string             `json:"type,omitempty" yaml:"type,omitempty"`
	Format      string             `json:"format,omitempty" yaml:"format,omitempty"`
	Description string             `json:"description,omitempty" yaml:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items       *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
	Required    []string           `json:"required,omitempty" yaml:"required,omitempty"`
	Example     interface{}        `json:"example,omitempty" yaml:"example,omitempty"`
	Enum        []interface{}      `json:"enum,omitempty" yaml:"enum,omitempty"`
	Ref         string             `json:"$ref,omitempty" yaml:"$ref,omitempty"`
}

// Components represents the components section
type Components struct {
	Schemas         map[string]*Schema        `json:"schemas,omitempty" yaml:"schemas,omitempty"`
	Responses       map[string]Response       `json:"responses,omitempty" yaml:"responses,omitempty"`
	Parameters      map[string]Parameter      `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBodies   map[string]RequestBody    `json:"requestBodies,omitempty" yaml:"requestBodies,omitempty"`
	Headers         map[string]Header         `json:"headers,omitempty" yaml:"headers,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty" yaml:"securitySchemes,omitempty"`
}

// SecurityRequirement represents a security requirement
type SecurityRequirement map[string][]string

// SecurityScheme represents a security scheme
type SecurityScheme struct {
	Type             string `json:"type" yaml:"type"`
	Description      string `json:"description,omitempty" yaml:"description,omitempty"`
	Name             string `json:"name,omitempty" yaml:"name,omitempty"`
	In               string `json:"in,omitempty" yaml:"in,omitempty"`
	Scheme           string `json:"scheme,omitempty" yaml:"scheme,omitempty"`
	BearerFormat     string `json:"bearerFormat,omitempty" yaml:"bearerFormat,omitempty"`
	Flows            *Flows `json:"flows,omitempty" yaml:"flows,omitempty"`
	OpenIDConnectURL string `json:"openIdConnectUrl,omitempty" yaml:"openIdConnectUrl,omitempty"`
}

// Flows represents OAuth2 flows
type Flows struct {
	Implicit          *Flow `json:"implicit,omitempty" yaml:"implicit,omitempty"`
	Password          *Flow `json:"password,omitempty" yaml:"password,omitempty"`
	ClientCredentials *Flow `json:"clientCredentials,omitempty" yaml:"clientCredentials,omitempty"`
	AuthorizationCode *Flow `json:"authorizationCode,omitempty" yaml:"authorizationCode,omitempty"`
}

// Flow represents an OAuth2 flow
type Flow struct {
	AuthorizationURL string            `json:"authorizationUrl,omitempty" yaml:"authorizationUrl,omitempty"`
	TokenURL         string            `json:"tokenUrl,omitempty" yaml:"tokenUrl,omitempty"`
	RefreshURL       string            `json:"refreshUrl,omitempty" yaml:"refreshUrl,omitempty"`
	Scopes           map[string]string `json:"scopes" yaml:"scopes"`
}

// Tag represents a tag
type Tag struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// ParseFile parses an OpenAPI/Swagger file from disk
func (p *Parser) ParseFile(filename string) (*OpenAPISpec, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.Parse(data, filename)
}

// Parse parses OpenAPI/Swagger data from bytes
func (p *Parser) Parse(data []byte, filename string) (*OpenAPISpec, error) {
	var spec OpenAPISpec

	// Determine format based on file extension or content
	isJSON := strings.HasSuffix(strings.ToLower(filename), ".json") ||
		strings.HasPrefix(strings.TrimSpace(string(data)), "{")

	if isJSON {
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	} else {
		if err := yaml.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	}

	// Validate the spec
	if err := p.validateSpec(&spec); err != nil {
		return nil, fmt.Errorf("invalid OpenAPI specification: %w", err)
	}

	return &spec, nil
}

// validateSpec performs basic validation on the OpenAPI spec
func (p *Parser) validateSpec(spec *OpenAPISpec) error {
	// Check version
	if spec.OpenAPI == "" && spec.Swagger == "" {
		return fmt.Errorf("missing OpenAPI/Swagger version")
	}

	// Check info
	if spec.Info.Title == "" {
		return fmt.Errorf("missing info.title")
	}

	if spec.Info.Version == "" {
		return fmt.Errorf("missing info.version")
	}

	// Check paths
	if spec.Paths == nil || len(spec.Paths) == 0 {
		return fmt.Errorf("no paths defined")
	}

	return nil
}

// GetOperations returns all operations from the spec
func (p *Parser) GetOperations(spec *OpenAPISpec) []OperationInfo {
	var operations []OperationInfo

	for path, pathItem := range spec.Paths {
		operations = append(operations, p.extractOperations(path, pathItem)...)
	}

	return operations
}

// OperationInfo contains information about an operation
type OperationInfo struct {
	Path        string
	Method      string
	Operation   *Operation
	OperationID string
}

// extractOperations extracts all operations from a path item
func (p *Parser) extractOperations(path string, pathItem PathItem) []OperationInfo {
	var operations []OperationInfo

	methods := map[string]*Operation{
		"GET":     pathItem.Get,
		"POST":    pathItem.Post,
		"PUT":     pathItem.Put,
		"DELETE":  pathItem.Delete,
		"OPTIONS": pathItem.Options,
		"HEAD":    pathItem.Head,
		"PATCH":   pathItem.Patch,
		"TRACE":   pathItem.Trace,
	}

	for method, operation := range methods {
		if operation != nil {
			operationID := operation.OperationID
			if operationID == "" {
				operationID = fmt.Sprintf("%s_%s", strings.ToLower(method), strings.ReplaceAll(strings.Trim(path, "/"), "/", "_"))
			}

			operations = append(operations, OperationInfo{
				Path:        path,
				Method:      method,
				Operation:   operation,
				OperationID: operationID,
			})
		}
	}

	return operations
}
