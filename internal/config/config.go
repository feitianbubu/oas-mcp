package config

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	SwaggerFile    string         `yaml:"swagger_file" mapstructure:"swagger_file"`
	Server         Server         `yaml:"server" mapstructure:"server"`
	Upstream       Upstream       `yaml:"upstream" mapstructure:"upstream"`
	Auth           Auth           `yaml:"auth" mapstructure:"auth"`
	Logging        Logging        `yaml:"logging" mapstructure:"logging"`
	EndpointConfig EndpointConfig `yaml:"endpoint_config" mapstructure:"endpoint_config"`
}

// Server configuration for MCP server
type Server struct {
	Mode string `yaml:"mode" mapstructure:"mode"`
	Host string `yaml:"host" mapstructure:"host"`
	Port int    `yaml:"port" mapstructure:"port"`
}

// Upstream configuration for the target API
type Upstream struct {
	BaseURL string `yaml:"base_url" mapstructure:"base_url"`
	Timeout int    `yaml:"timeout" mapstructure:"timeout"`
}

// Auth configuration for authentication
type Auth struct {
	Type     string `yaml:"type" mapstructure:"type"`
	Token    string `yaml:"token" mapstructure:"token"`
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
	APIKey   string `yaml:"api_key" mapstructure:"api_key"`
}

// Logging configuration
type Logging struct {
	Level          string `yaml:"level" mapstructure:"level"`
	DisableConsole bool   `yaml:"disable_console" mapstructure:"disable_console"`
	File           string `yaml:"file" mapstructure:"file"`
}

// EndpointConfig holds endpoint-specific configuration
type EndpointConfig struct {
	DefaultTimeout int               `yaml:"default_timeout" mapstructure:"default_timeout"`
	Endpoints      map[string]string `yaml:"endpoints" mapstructure:"endpoints"`
}

// Server mode constants
const (
	ServerModeSTDIO = "stdio"
	ServerModeHTTP  = "http"
	ServerModeSSE   = "sse"
)

// InitFlags initializes command-line flags
func InitFlags() {
	pflag.StringP("config", "c", "", "Configuration file path")
	pflag.String("swagger-file", "swagger.json", "Path to the OpenAPI/Swagger file")
	pflag.String("mode", "stdio", "Server mode (stdio, http, sse)")
	pflag.String("host", "localhost", "Server host")
	pflag.Int("port", 8080, "Server port")
	pflag.String("upstream-base-url", "", "Upstream API base URL")
	pflag.Int("upstream-timeout", 30, "Upstream API timeout in seconds")
	pflag.String("auth-type", "none", "Authentication type (none, bearer, basic, apikey, oauth2)")
	pflag.String("auth-token", "", "Authentication token")
	pflag.String("auth-username", "", "Authentication username")
	pflag.String("auth-password", "", "Authentication password")
	pflag.String("auth-api-key", "", "Authentication API key")
	pflag.String("log-level", "info", "Log level")
	pflag.Bool("log-disable-console", false, "Disable console logging")
	pflag.String("log-file", "", "Log file path")

	// Bind flags to viper
	viper.BindPFlag("swagger_file", pflag.Lookup("swagger-file"))
	viper.BindPFlag("server.mode", pflag.Lookup("mode"))
	viper.BindPFlag("server.host", pflag.Lookup("host"))
	viper.BindPFlag("server.port", pflag.Lookup("port"))
	viper.BindPFlag("upstream.base_url", pflag.Lookup("upstream-base-url"))
	viper.BindPFlag("upstream.timeout", pflag.Lookup("upstream-timeout"))
	viper.BindPFlag("auth.type", pflag.Lookup("auth-type"))
	viper.BindPFlag("auth.token", pflag.Lookup("auth-token"))
	viper.BindPFlag("auth.username", pflag.Lookup("auth-username"))
	viper.BindPFlag("auth.password", pflag.Lookup("auth-password"))
	viper.BindPFlag("auth.api_key", pflag.Lookup("auth-api-key"))
	viper.BindPFlag("logging.level", pflag.Lookup("log-level"))
	viper.BindPFlag("logging.disable_console", pflag.Lookup("log-disable-console"))
	viper.BindPFlag("logging.file", pflag.Lookup("log-file"))

	// Set environment variable prefix
	viper.SetEnvPrefix("OAS_MCP")
	viper.AutomaticEnv()
}

// Load loads the configuration from viper
func Load() (*Config, error) {
	var cfg Config

	// Set defaults
	setDefaults()

	// Read config file if specified
	if configFile := pflag.Lookup("config").Value.String(); configFile != "" {
		viper.SetConfigFile(configFile)
		if err := viper.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		// Look for config files in current directory
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.ReadInConfig() // Ignore error if config file not found
	}

	// Unmarshal configuration
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Check if swagger file exists
	if c.SwaggerFile == "" {
		return fmt.Errorf("swagger_file is required")
	}

	if _, err := os.Stat(c.SwaggerFile); os.IsNotExist(err) {
		return fmt.Errorf("swagger file does not exist: %s", c.SwaggerFile)
	}

	// Validate server mode
	validModes := map[string]bool{
		"stdio": true,
		"http":  true,
		"sse":   true,
	}

	if !validModes[c.Server.Mode] {
		return fmt.Errorf("invalid server mode: %s (must be stdio, http, or sse)", c.Server.Mode)
	}

	// Validate auth type
	validAuthTypes := map[string]bool{
		"none":   true,
		"bearer": true,
		"basic":  true,
		"apikey": true,
		"oauth2": true,
	}

	if !validAuthTypes[c.Auth.Type] {
		return fmt.Errorf("invalid auth type: %s", c.Auth.Type)
	}

	// Validate auth configuration based on type
	switch c.Auth.Type {
	case "bearer", "apikey":
		if c.Auth.Token == "" && c.Auth.APIKey == "" {
			return fmt.Errorf("token or api_key is required for %s auth", c.Auth.Type)
		}
	case "basic":
		if c.Auth.Username == "" || c.Auth.Password == "" {
			return fmt.Errorf("username and password are required for basic auth")
		}
	}

	return nil
}

// GetVersionInfo returns version information
func GetVersionInfo() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "oas-mcp (unknown version)"
	}

	version := "unknown"
	commit := "unknown"
	date := "unknown"

	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}

	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			if len(setting.Value) >= 7 {
				commit = setting.Value[:7]
			}
		case "vcs.time":
			date = setting.Value
		}
	}

	return fmt.Sprintf("oas-mcp version %s (commit: %s, built: %s)", version, commit, date)
}

// SaveExample saves an example configuration file
func SaveExample(filename string) error {
	cfg := &Config{
		SwaggerFile: "swagger.json",
		Server: Server{
			Mode: "stdio",
			Host: "localhost",
			Port: 8080,
		},
		Upstream: Upstream{
			BaseURL: "https://api.example.com",
			Timeout: 30,
		},
		Auth: Auth{
			Type:     "none",
			Token:    "",
			Username: "",
			Password: "",
			APIKey:   "",
		},
		Logging: Logging{
			Level:          "info",
			DisableConsole: false,
			File:           "",
		},
		EndpointConfig: EndpointConfig{
			DefaultTimeout: 30,
			Endpoints:      make(map[string]string),
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// setDefaults sets default configuration values
func setDefaults() {
	viper.SetDefault("swagger_file", "swagger.json")
	viper.SetDefault("server.mode", "stdio")
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("upstream.base_url", "")
	viper.SetDefault("upstream.timeout", 30)
	viper.SetDefault("auth.type", "none")
	viper.SetDefault("auth.token", "")
	viper.SetDefault("auth.username", "")
	viper.SetDefault("auth.password", "")
	viper.SetDefault("auth.api_key", "")
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.disable_console", false)
	viper.SetDefault("logging.file", "")
	viper.SetDefault("endpoint_config.default_timeout", 30)
}
