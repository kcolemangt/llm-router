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
		LLMRouterAPIKeyEnv: "LLMROUTER_API_KEY",
		Aliases:            make(map[string]string),
	}

	// Initialize command-line flags
	configFile, llmRouterAPIKeyEnv, llmRouterAPIKey, listeningPort, logLevel := config.InitFlags()

	// Initialize the logger
	logger, err := logging.NewLogger(logLevel)
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// Load the configuration
	cfg, err := config.LoadConfig(configFile, llmRouterAPIKeyEnv, llmRouterAPIKey, listeningPort, defaultConfig, logger)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// If using a generated key, print a helpful message
	if cfg.UseGeneratedKey {
		fmt.Printf(`
Your LLM-Router endpoint will be exposed publicly so that Cursor's servers can invoke it.
A strong API key is highly recommended to prevent others from consuming your resources.

You may specify the API key via:
- Environment variable: export %s=your_api_key
- Command line flag: --llmrouter-api-key=your_api_key

Since neither of those have been set, we've generated a unique key for this session:
%s

This is what you should set as your API key in Cursor.
`, cfg.LLMRouterAPIKeyEnv, cfg.LLMRouterAPIKey)
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
