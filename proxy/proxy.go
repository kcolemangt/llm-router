package proxy

import (
	"fmt"
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

		// Add custom transport to log responses
		originalTransport := http.DefaultTransport
		proxy.Transport = &debugTransport{
			transport: originalTransport,
			logger:    logger,
			backend:   backend.Name,
		}

		Proxies[strings.TrimSpace(backend.Prefix)] = proxy
		if backend.Default {
			DefaultProxy = proxy
			logger.Debug("Default proxy set", zap.String("backend", backend.Name))
		}
	}
}

// debugTransport is a custom http.RoundTripper that logs request and response details
type debugTransport struct {
	transport http.RoundTripper
	logger    *zap.Logger
	backend   string
}

// RoundTrip implements the http.RoundTripper interface
func (t *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request body for logging
	var reqBodyStr string
	if req.Body != nil {
		req.Body, reqBodyStr = utils.DrainBody(req.Body)
	}

	// Log the full request if debug is enabled
	t.logger.Debug("Outgoing request to backend",
		zap.String("backend", t.backend),
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()))

	// Log all headers being sent to backend (at debug level)
	for name, values := range req.Header {
		if strings.ToLower(name) == "authorization" {
			t.logger.Debug("Outgoing header",
				zap.String("name", name),
				zap.String("value", utils.RedactAuthorization(values[0])))
		} else {
			t.logger.Debug("Outgoing header",
				zap.String("name", name),
				zap.String("value", strings.Join(values, ", ")))
		}
	}

	// Execute the request
	resp, err := t.transport.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	// Log all response headers from backend (at debug level)
	t.logger.Debug("Response headers from backend")
	for name, values := range resp.Header {
		t.logger.Debug("Response header",
			zap.String("name", name),
			zap.String("value", strings.Join(values, ", ")))
	}

	// Check if this is a streaming response
	isStreaming := false
	if resp != nil {
		contentType := resp.Header.Get("Content-Type")
		transferEncoding := resp.Header.Get("Transfer-Encoding")
		isStreaming = strings.Contains(contentType, "text/event-stream") ||
			transferEncoding == "chunked" ||
			(req.URL.Path == "/chat/completions" && strings.Contains(reqBodyStr, "\"stream\":true"))
	}

	// Clone the response body for logging, preserving streaming if needed
	var respBodyStr string
	if resp.Body != nil {
		resp.Body, respBodyStr = utils.DrainAndCapture(resp.Body, isStreaming)
	}

	// Log request and response details, being careful with streaming content
	if isStreaming {
		t.logger.Debug("Streaming response detected - logging headers only",
			zap.Int("status", resp.StatusCode),
			zap.String("contentType", resp.Header.Get("Content-Type")),
			zap.String("transferEncoding", resp.Header.Get("Transfer-Encoding")))
	} else {
		utils.LogRequestResponse(t.logger, req, resp, reqBodyStr, respBodyStr)
	}

	return resp, nil
}

// makeDirector returns a function that modifies requests to route through the reverse proxy
func makeDirector(urlParsed *url.URL, backend model.BackendConfig, logger *zap.Logger) func(req *http.Request) {
	return func(req *http.Request) {
		originalHost := req.Host
		originalPath := req.URL.Path

		// Set target host and URL
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

		// Set standard proxy headers
		req.Header.Set("Host", urlParsed.Host)

		// Extract client IP from RemoteAddr properly
		clientIP := req.RemoteAddr
		if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
			clientIP = clientIP[:idx]
		}
		clientIP = strings.Trim(clientIP, "[]")

		// Set standard proxy headers
		req.Header.Set("X-Real-IP", clientIP)

		// Handle X-Forwarded-For
		if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
			req.Header.Set("X-Forwarded-For", fmt.Sprintf("%s, %s", xff, clientIP))
		} else {
			req.Header.Set("X-Forwarded-For", clientIP)
		}

		// Set X-Forwarded-Proto
		req.Header.Set("X-Forwarded-Proto", "https")

		// Set X-Forwarded-Host
		req.Header.Set("X-Forwarded-Host", originalHost)

		// Handle authentication based on backend config
		if backend.RequireAPIKey {
			var apiKey string

			// First try to get API key from backend-specific environment variable
			if backend.KeyEnvVar != "" {
				apiKey = os.Getenv(backend.KeyEnvVar)
			}

			// If not found and this is OpenAI backend, try to use the global API key
			if apiKey == "" && backend.Name == "openai" {
				apiKey = os.Getenv("OPENAI_API_KEY")
			}

			if apiKey != "" {
				auth := "Bearer " + apiKey
				req.Header.Set("Authorization", auth)
				logger.Info("Set Authorization header using API key",
					zap.String("backend", backend.Name),
					zap.String("source", backend.KeyEnvVar),
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
					logger.Error("Missing required API key for backend",
						zap.String("backend", backend.Name),
						zap.String("envVar", backend.KeyEnvVar),
					)
					// We'll let the actual backend API respond with the appropriate error
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
