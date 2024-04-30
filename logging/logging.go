package logging

import (
	"go.uber.org/zap"
)

// NewLogger initializes and returns a new zap.Logger based on the provided log level.
func NewLogger(level string) (*zap.Logger, error) {
	var zapConfig zap.Config

	// Set up production or development config based on your needs
	if level == "debug" {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	// Adjust log level based on input
	var logLevel zap.AtomicLevel
	err := logLevel.UnmarshalText([]byte(level))
	if err != nil {
		return nil, err
	}
	zapConfig.Level = logLevel

	// Build and return the configured logger
	logger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}
