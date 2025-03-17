package model

import "go.uber.org/zap"

// BackendConfig defines the structure for backend configuration
type BackendConfig struct {
	Name              string            `json:"name"`
	BaseURL           string            `json:"base_url"`
	Prefix            string            `json:"prefix"`
	Default           bool              `json:"default"`
	RequireAPIKey     bool              `json:"require_api_key"`
	KeyEnvVar         string            `json:"key_env_var"`
	RoleRewrites      map[string]string `json:"role_rewrites,omitempty"`
	UnsupportedParams []string          `json:"unsupported_params,omitempty"`
}

// Config is the structure for the proxy configuration
type Config struct {
	ListeningPort      int `json:"listening_port"`
	Logger             *zap.Logger
	Backends           []BackendConfig `json:"backends"`
	LLMRouterAPIKeyEnv string          `json:"llmrouter_api_key_env"`
	LLMRouterAPIKey    string
	UseGeneratedKey    bool
	Aliases            map[string]string `json:"aliases"`
}
