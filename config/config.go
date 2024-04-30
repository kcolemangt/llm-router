package config

import (
	"encoding/json"
	"flag"
	"os"

	"github.com/kcolemangt/llm-router/model"
	"github.com/kcolemangt/llm-router/utils"
	"go.uber.org/zap"
)

// LoadConfig loads the configuration from the specified file or from a default if the file cannot be read.
func LoadConfig(configFile, apiKeyEnvVar string, listeningPort int, defaultConfig model.Config, logger *zap.Logger) (*model.Config, error) {
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
	if apiKeyEnvVar != "" {
		cfg.GlobalAPIKeyEnv = apiKeyEnvVar
		logger.Info("API key environment variable override applied", zap.String("APIKeyEnvVar", apiKeyEnvVar))
	}
	if listeningPort != 0 {
		cfg.ListeningPort = listeningPort
		logger.Info("Listening port override applied", zap.Int("port", listeningPort))
	}

	cfg.Logger = logger

	cfg.GlobalAPIKey = os.Getenv(cfg.GlobalAPIKeyEnv)
	if cfg.GlobalAPIKey == "" {
		logger.Fatal("API key environment variable not set", zap.String("variable", cfg.GlobalAPIKeyEnv))
	} else {
		logger.Info("API key retrieved from environment variable", zap.String("APIKey", utils.RedactAuthorization(cfg.GlobalAPIKey)))
	}

	logger.Info("Configuration loading completed successfully")
	return &cfg, nil
}

// InitFlags initializes and parses the command-line flags.
func InitFlags() (string, string, int, string) {
	configFile := flag.String("config", "config.json", "Path to the configuration file")
	apiKeyEnvVar := flag.String("api-key-env", "OPENAI_API_KEY", "Environment variable for the API key (overrides config file)")
	listeningPort := flag.Int("port", 0, "Listening port (overrides config file)")
	logLevel := flag.String("log-level", "warn", "define the log level: debug, info, warn, error, dpanic, panic, fatal")

	flag.Parse()

	return *configFile, *apiKeyEnvVar, *listeningPort, *logLevel
}
