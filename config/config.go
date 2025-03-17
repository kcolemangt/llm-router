package config

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/joho/godotenv"
	"github.com/kcolemangt/llm-router/model"
	"github.com/kcolemangt/llm-router/utils"
	"go.uber.org/zap"
)

// LoadConfig loads the configuration from the specified file or from a default if the file cannot be read.
func LoadConfig(configFile, llmRouterAPIKeyEnv, llmRouterAPIKey string, listeningPort int, defaultConfig model.Config, logger *zap.Logger) (*model.Config, error) {
	// Load environment variables from .env file if it exists
	// We use godotenv's Load function which respects the precedence where existing environment
	// variables take priority over values defined in the .env file
	if err := godotenv.Load(); err != nil {
		logger.Debug("No .env file found or unable to load it, continuing with system environment variables", zap.Error(err))
	} else {
		logger.Info(".env file loaded successfully")
	}

	// Start of configuration loading
	logger.Info("Starting configuration loading", zap.String("configFile", configFile))

	var cfg model.Config
	if _, err := os.Stat(configFile); err == nil { // If the file exists
		logger.Info("Config file found", zap.String("file", configFile))
		fileData, err := os.ReadFile(configFile)
		if err != nil {
			logger.Error("Failed to read config file", zap.String("file", configFile), zap.Error(err))
			return nil, err
		}
		err = json.Unmarshal(fileData, &cfg) // Unmarshal the JSON data into the Config struct
		if err != nil {
			logger.Error("Failed to unmarshal config data", zap.String("file", configFile), zap.Error(err))
			return nil, err
		}
		logger.Info("Config file loaded and parsed", zap.String("file", configFile))
	} else { // If the file doesn't exist, use the default config
		logger.Warn("Config file not found, using default configuration", zap.String("file", configFile))
		cfg = defaultConfig
	}

	// Apply command line overrides
	if listeningPort != 0 {
		cfg.ListeningPort = listeningPort
		logger.Info("Listening port override applied", zap.Int("port", listeningPort))
	}

	// Set LLM Router API key environment variable
	if llmRouterAPIKeyEnv != "" {
		cfg.LLMRouterAPIKeyEnv = llmRouterAPIKeyEnv
		logger.Info("LLM Router API key environment variable override applied", zap.String("LLMRouterAPIKeyEnv", llmRouterAPIKeyEnv))
	} else if cfg.LLMRouterAPIKeyEnv == "" {
		cfg.LLMRouterAPIKeyEnv = "LLMROUTER_API_KEY"
		logger.Info("Using default LLM Router API key environment variable", zap.String("LLMRouterAPIKeyEnv", cfg.LLMRouterAPIKeyEnv))
	}

	// Try to get the LLM Router API key from environment or command line
	if llmRouterAPIKey != "" {
		// Use the API key provided via command line flag
		cfg.LLMRouterAPIKey = llmRouterAPIKey
		logger.Info("Using LLM Router API key from command line", zap.String("LLMRouterAPIKey", utils.RedactAuthorization(cfg.LLMRouterAPIKey)))
	} else if envAPIKey := os.Getenv(cfg.LLMRouterAPIKeyEnv); envAPIKey != "" {
		// Use the API key from environment variable
		cfg.LLMRouterAPIKey = envAPIKey
		logger.Info("Using LLM Router API key from environment variable", zap.String("LLMRouterAPIKey", utils.RedactAuthorization(cfg.LLMRouterAPIKey)))
	} else {
		// Generate a random API key for this session
		generatedKey, err := utils.GenerateStrongAPIKey()
		if err != nil {
			logger.Error("Failed to generate LLM Router API key", zap.Error(err))
			return nil, err
		}
		cfg.LLMRouterAPIKey = generatedKey
		cfg.UseGeneratedKey = true
		logger.Info("Generated LLM Router API key for this session", zap.String("LLMRouterAPIKey", utils.RedactAuthorization(cfg.LLMRouterAPIKey)))
	}

	cfg.Logger = logger

	logger.Info("Configuration loading completed successfully")
	return &cfg, nil
}

// InitFlags initializes and parses the command-line flags.
func InitFlags() (string, string, string, int, string) {
	configFile := flag.String("config", "config.json", "Path to the configuration file")
	llmRouterAPIKeyEnv := flag.String("llmrouter-api-key-env", "LLMROUTER_API_KEY", "Environment variable for the LLM Router API key")
	llmRouterAPIKey := flag.String("llmrouter-api-key", "", "LLM Router API key to use (takes precedence over environment variable)")
	listeningPort := flag.Int("port", 0, "Listening port (overrides config file)")
	logLevel := flag.String("log-level", "warn", "define the log level: debug, info, warn, error, dpanic, panic, fatal")

	flag.Parse()

	return *configFile, *llmRouterAPIKeyEnv, *llmRouterAPIKey, *listeningPort, *logLevel
}
