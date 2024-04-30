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
		ListeningPort:   8080,
		GlobalAPIKeyEnv: "TEST_API_KEY", // Set the environment variable key
	}

	// Set a dummy API key environment variable to prevent fatal error
	os.Setenv("TEST_API_KEY", "dummy_api_key")
	defer os.Unsetenv("TEST_API_KEY") // Clean up after the test

	// Simulate missing file scenario by passing a non-existent file name
	config, err := LoadConfig("non_existent_config.json", "", 0, defaultConfig, logger)
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

	config, err := LoadConfig("test_config.json", "NEW_API_KEY", 8080, defaultConfig, logger)
	if err != nil {
		t.Errorf("Failed to load config with overrides: %s", err)
	}

	if config.ListeningPort != 8080 {
		t.Errorf("Expected overridden ListeningPort 8080, got %d", config.ListeningPort)
	}

	if config.GlobalAPIKeyEnv != "NEW_API_KEY" {
		t.Errorf("Expected APIKeyEnvVar 'NEW_API_KEY', got '%s'", config.GlobalAPIKeyEnv)
	}

	if config.GlobalAPIKey != "test_api_key" {
		t.Errorf("Expected GlobalAPIKey 'test_api_key', got '%s'", config.GlobalAPIKey)
	}
}

func TestAPIKeyEnvVariable(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defaultConfig := model.Config{}

	os.Setenv("TEST_API_KEY", "12345")
	config, err := LoadConfig("test_config.json", "TEST_API_KEY", 0, defaultConfig, logger)
	if err != nil {
		t.Errorf("Failed to load config with API key env: %s", err)
	}

	if config.GlobalAPIKey != "12345" {
		t.Errorf("Expected GlobalAPIKey '12345', got '%s'", config.GlobalAPIKey)
	}
	os.Unsetenv("TEST_API_KEY")
}

func TestErrorReadingFile(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defaultConfig := model.Config{
		ListeningPort:   8080,
		GlobalAPIKey:    "dummy_value", // Ensure this matches what LoadConfig sets when no file is found
		GlobalAPIKeyEnv: "DUMMY_API_KEY",
		// Add other fields with expected default values here
	}

	// Set a dummy API key environment variable to prevent fatal error
	os.Setenv("DUMMY_API_KEY", "dummy_value")
	defer os.Unsetenv("DUMMY_API_KEY") // Clean up after the test

	// Generate an invalid file path that should be invalid on any OS
	invalidFilePath := filepath.Join(os.TempDir(), "non_existent_directory", "non_existent_file.json")

	config, err := LoadConfig(invalidFilePath, "DUMMY_API_KEY", 0, defaultConfig, logger)
	if err != nil {
		t.Errorf("Did not expect an error, but got: %s", err)
	}

	// Check if the returned config matches the default config
	if config.ListeningPort != defaultConfig.ListeningPort || config.GlobalAPIKey != defaultConfig.GlobalAPIKey || config.GlobalAPIKeyEnv != defaultConfig.GlobalAPIKeyEnv {
		t.Errorf("Expected default configuration, got: %+v", config)
	}
}
