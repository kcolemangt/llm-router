package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/kcolemangt/llm-router/model"
	"github.com/kcolemangt/llm-router/utils"
	"go.uber.org/zap"
)

// Proxies holds the created reverse proxies by prefix
var Proxies map[string]*httputil.ReverseProxy

// DefaultProxy is the default reverse proxy used when no specific match is found
var DefaultProxy *httputil.ReverseProxy

// InitializeProxies sets up the reverse proxy handlers based on the backend configurations
func InitializeProxies(backends []model.BackendConfig, logger *zap.Logger) {
	Proxies = make(map[string]*httputil.ReverseProxy)

	for _, backend := range backends {
		urlParsed, err := url.Parse(backend.BaseURL)
		if err != nil {
			logger.Fatal("Error parsing URL for backend", zap.String("backend", backend.Name), zap.Error(err))
		}

		proxy := httputil.NewSingleHostReverseProxy(urlParsed)
		proxy.Director = makeDirector(urlParsed, backend, logger)

		Proxies[strings.TrimSpace(backend.Prefix)] = proxy
		if backend.Default {
			DefaultProxy = proxy
			logger.Debug("Default proxy set", zap.String("backend", backend.Name))
		}
	}
}

// makeDirector returns a function that modifies requests to route through the reverse proxy
func makeDirector(urlParsed *url.URL, backend model.BackendConfig, logger *zap.Logger) func(req *http.Request) {
	return func(req *http.Request) {
		originalHost := req.Host
		originalPath := req.URL.Path
		req.Host = urlParsed.Host
		req.URL.Scheme = urlParsed.Scheme
		req.URL.Host = urlParsed.Host
		req.URL.Path = urlParsed.Path + originalPath

		// Log the modifications to the request URL and Host
		logger.Info("Modified request URL and Host",
			zap.String("originalHost", originalHost),
			zap.String("newHost", req.Host),
			zap.String("originalPath", originalPath),
			zap.String("newPath", req.URL.Path),
		)

		req.Header.Set("X-Forwarded-Host", originalHost)
		logger.Debug("Set X-Forwarded-Host header", zap.String("X-Forwarded-Host", originalHost))

		if backend.RequireAPIKey {
			apiKey := os.Getenv(backend.KeyEnvVar)
			if apiKey != "" {
				auth := "Bearer " + apiKey
				req.Header.Set("Authorization", auth)
				logger.Info("Set Authorization header using API key",
					zap.String("backend", backend.Name),
					zap.String("APIKeyEnvVar", backend.KeyEnvVar),
					zap.String("Authorization", utils.RedactAuthorization(auth)),
				)
			} else {
				existingAuth := req.Header.Get("Authorization")
				if existingAuth != "" {
					logger.Info("Authorization header already set, forwarding to backend",
						zap.String("backend", backend.Name),
						zap.String("Authorization", utils.RedactAuthorization(existingAuth)),
					)
				} else {
					logger.Fatal("Missing required API key for backend, rejecting request",
						zap.String("backend", backend.Name),
						zap.String("envVar", backend.KeyEnvVar),
					)
				}
			}
		} else {
			req.Header.Del("Authorization")
			logger.Info("Removed Authorization header for backend", zap.String("backend", backend.Name))
		}

		logger.Info("Proxy Director handled request",
			zap.String("URL", req.URL.String()),
			zap.String("Host", req.Host),
			zap.String("Method", req.Method),
			zap.String("Protocol", req.Proto),
		)
	}
}
