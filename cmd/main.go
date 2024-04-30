package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/kcolemangt/llm-router/config"
	"github.com/kcolemangt/llm-router/handler"
	"github.com/kcolemangt/llm-router/logging"
	"github.com/kcolemangt/llm-router/model"
	"github.com/kcolemangt/llm-router/proxy"
	"go.uber.org/zap"
)

func main() {
	// DefaultConfig is the default configuration in case the configuration file cannot be read.
	var defaultConfig = model.Config{
		ListeningPort: 11411,
		Backends: []model.BackendConfig{
			{
				Name:          "openai",
				BaseURL:       "https://api.openai.com",
				Prefix:        "openai/",
				Default:       true,
				RequireAPIKey: true,
			},
			{
				Name:    "ollama",
				BaseURL: "http://localhost:11434",
				Prefix:  "ollama/",
			},
		},
	}

	// Initialize command-line flags
	configFile, apiKeyEnvVar, listeningPort, logLevel := config.InitFlags()

	// Initialize the logger
	logger, err := logging.NewLogger(logLevel)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// Load the configuration
	cfg, err := config.LoadConfig(configFile, apiKeyEnvVar, listeningPort, defaultConfig, logger)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Initialize proxies based on the loaded configuration
	proxy.InitializeProxies(cfg.Backends, logger)

	// Set up HTTP server and handlers
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler.HandleRequest(cfg, w, r)
	})

	// Start the server
	addr := fmt.Sprintf(":%d", cfg.ListeningPort)
	log.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}
}
