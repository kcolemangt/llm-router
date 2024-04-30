package proxy

import (
	"testing"

	"github.com/kcolemangt/llm-router/model"
	"go.uber.org/zap"
)

func TestMultipleProxiesInitialization(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	backends := []model.BackendConfig{
		{Name: "test1", BaseURL: "http://localhost:8081", Prefix: "test1/"},
		{Name: "test2", BaseURL: "http://localhost:8082", Prefix: "test2/", Default: true},
	}

	InitializeProxies(backends, logger)
	if len(Proxies) != 2 {
		t.Errorf("Expected 2 proxies, got %d", len(Proxies))
	}
	if DefaultProxy != Proxies["test2/"] {
		t.Errorf("Default proxy not set correctly")
	}
}
