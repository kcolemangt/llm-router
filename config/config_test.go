package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kcolemangt/llm-router/model"
	"go.uber.org/zap"
)

func TestMissingConfigFile(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defaultConfig := model.Config{
		ListeningPort:      8080,
		LLMRouterAPIKeyEnv: "TEST_API_KEY", // Set the environment variable key
	}

	// Set a dummy API key environment variable
	os.Setenv("TEST_API_KEY", "dummy_api_key")
	defer os.Unsetenv("TEST_API_KEY") // Clean up after the test

	// Simulate missing file scenario by passing a non-existent file name
	config, err := LoadConfig("non_existent_config.json", "TEST_API_KEY", "", 0, defaultConfig, logger)
	if err != nil {
		t.Errorf("Failed to handle missing config file: %s", err)
	}

	if config.ListeningPort != 8080 {
		t.Errorf("Expected default ListeningPort 8080, got %d", config.ListeningPort)
	}
}

func TestCommandLineOverrides(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defaultConfig := model.Config{
		ListeningPort: 11411,
	}

	// Set the environment variable expected by the test
	os.Setenv("NEW_API_KEY", "test_api_key")
	defer os.Unsetenv("NEW_API_KEY") // Clean up after the test

	config, err := LoadConfig("test_config.json", "NEW_API_KEY", "", 8080, defaultConfig, logger)
	if err != nil {
		t.Errorf("Failed to load config with overrides: %s", err)
	}

	if config.ListeningPort != 8080 {
		t.Errorf("Expected overridden ListeningPort 8080, got %d", config.ListeningPort)
	}

	if config.LLMRouterAPIKeyEnv != "NEW_API_KEY" {
		t.Errorf("Expected LLMRouterAPIKeyEnv 'NEW_API_KEY', got '%s'", config.LLMRouterAPIKeyEnv)
	}

	if config.LLMRouterAPIKey != "test_api_key" {
		t.Errorf("Expected LLMRouterAPIKey 'test_api_key', got '%s'", config.LLMRouterAPIKey)
	}
}

func TestAPIKeyEnvVariable(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defaultConfig := model.Config{}

	os.Setenv("TEST_API_KEY", "12345")
	config, err := LoadConfig("test_config.json", "TEST_API_KEY", "", 0, defaultConfig, logger)
	if err != nil {
		t.Errorf("Failed to load config with API key env: %s", err)
	}

	if config.LLMRouterAPIKey != "12345" {
		t.Errorf("Expected LLMRouterAPIKey '12345', got '%s'", config.LLMRouterAPIKey)
	}
	os.Unsetenv("TEST_API_KEY")
}

func TestCommandLineAPIKey(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defaultConfig := model.Config{}

	// Test with command line API key, which should take precedence
	config, err := LoadConfig("test_config.json", "TEST_API_KEY", "command_line_key", 0, defaultConfig, logger)
	if err != nil {
		t.Errorf("Failed to load config with command line API key: %s", err)
	}

	if config.LLMRouterAPIKey != "command_line_key" {
		t.Errorf("Expected LLMRouterAPIKey 'command_line_key', got '%s'", config.LLMRouterAPIKey)
	}
}

func TestErrorReadingFile(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defaultConfig := model.Config{
		ListeningPort:      8080,
		LLMRouterAPIKeyEnv: "DUMMY_API_KEY",
		// Add other fields with expected default values here
	}

	// Set a dummy API key environment variable
	os.Setenv("DUMMY_API_KEY", "dummy_value")
	defer os.Unsetenv("DUMMY_API_KEY") // Clean up after the test

	// Generate an invalid file path that should be invalid on any OS
	invalidFilePath := filepath.Join(os.TempDir(), "non_existent_directory", "non_existent_file.json")

	config, err := LoadConfig(invalidFilePath, "DUMMY_API_KEY", "", 0, defaultConfig, logger)
	if err != nil {
		t.Errorf("Did not expect an error, but got: %s", err)
	}

	// Check if the returned config matches the default config
	if config.ListeningPort != defaultConfig.ListeningPort || config.LLMRouterAPIKeyEnv != defaultConfig.LLMRouterAPIKeyEnv {
		t.Errorf("Expected default configuration, got: %+v", config)
	}
}

func TestGeneratedAPIKey(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defaultConfig := model.Config{
		LLMRouterAPIKeyEnv: "NONEXISTENT_ENV_VAR",
	}

	// Make sure the environment variable doesn't exist
	os.Unsetenv("NONEXISTENT_ENV_VAR")

	config, err := LoadConfig("test_config.json", "", "", 0, defaultConfig, logger)
	if err != nil {
		t.Errorf("Failed to load config with generated API key: %s", err)
	}

	// Verify a key was generated
	if config.LLMRouterAPIKey == "" {
		t.Errorf("Expected a generated API key, got empty string")
	}

	// Check it has the correct prefix
	if len(config.LLMRouterAPIKey) < 4 || config.LLMRouterAPIKey[:4] != "rsk_" {
		t.Errorf("Expected API key with 'rsk_' prefix, got: %s", config.LLMRouterAPIKey)
	}

	// Verify UseGeneratedKey flag is set
	if !config.UseGeneratedKey {
		t.Errorf("Expected UseGeneratedKey to be true for generated keys")
	}
}

func TestDotEnvLoading(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defaultConfig := model.Config{
		LLMRouterAPIKeyEnv: "ENV_TEST_KEY",
	}

	// Create a temporary .env file
	tmpDir := t.TempDir()
	envFilePath := filepath.Join(tmpDir, ".env")
	err := os.WriteFile(envFilePath, []byte("ENV_TEST_KEY=from_dotenv\nTEST_VARIABLE=test_value"), 0644)
	if err != nil {
		t.Fatalf("Failed to create temporary .env file: %s", err)
	}

	// Set the working directory to the temporary directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %s", err)
	}
	defer os.Chdir(originalWd) // Restore the original working directory

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change working directory: %s", err)
	}

	// Case 1: Test loading from .env file when no environment variable is set
	os.Unsetenv("ENV_TEST_KEY")
	config, err := LoadConfig("nonexistent_config.json", "ENV_TEST_KEY", "", 0, defaultConfig, logger)
	if err != nil {
		t.Errorf("Failed to load config with .env file: %s", err)
	}

	if config.LLMRouterAPIKey != "from_dotenv" {
		t.Errorf("Expected LLMRouterAPIKey 'from_dotenv', got '%s'", config.LLMRouterAPIKey)
	}

	// Case 2: Test precedence where environment variable overrides .env file
	os.Setenv("ENV_TEST_KEY", "from_environment")
	config, err = LoadConfig("nonexistent_config.json", "ENV_TEST_KEY", "", 0, defaultConfig, logger)
	if err != nil {
		t.Errorf("Failed to load config with environment override: %s", err)
	}

	if config.LLMRouterAPIKey != "from_environment" {
		t.Errorf("Expected LLMRouterAPIKey 'from_environment', got '%s'", config.LLMRouterAPIKey)
	}

	// Clean up
	os.Unsetenv("ENV_TEST_KEY")
}
