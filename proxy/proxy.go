package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

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

		// Create a proxy with standard settings
		proxy := httputil.NewSingleHostReverseProxy(urlParsed)

		// Set up our custom director
		proxy.Director = makeDirector(urlParsed, backend, logger)

		// Configure error handler to provide better error messages
		proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
			logger.Error("Proxy error",
				zap.String("backend", backend.Name),
				zap.String("url", req.URL.String()),
				zap.Error(err))

			// Return a proper error to the client
			http.Error(rw, fmt.Sprintf("Error communicating with backend service: %v", err), http.StatusBadGateway)
		}

		// Add custom transport to log requests/responses
		originalTransport := http.DefaultTransport.(*http.Transport).Clone()

		// Adjust transport timeouts
		originalTransport.ResponseHeaderTimeout = 30 * time.Second
		originalTransport.TLSHandshakeTimeout = 10 * time.Second
		originalTransport.ExpectContinueTimeout = 5 * time.Second
		originalTransport.MaxIdleConns = 100
		originalTransport.MaxConnsPerHost = 20
		originalTransport.MaxIdleConnsPerHost = 10

		// Wrap transport in our debug transport for logging
		proxy.Transport = &debugTransport{
			transport: originalTransport,
			logger:    logger,
			backend:   backend.Name,
		}

		// Add to our proxies map
		Proxies[strings.TrimSpace(backend.Prefix)] = proxy

		// Set default if configured
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
	// Clone the request body for logging without altering the original request
	var reqBodyStr string
	var bodyBytes []byte
	if req.Body != nil {
		// Read the body
		bodyBytes, _ = io.ReadAll(req.Body)
		// Recreate the body exactly as it was
		req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Format for logging only
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, bodyBytes, "", "  "); err == nil {
			reqBodyStr = prettyJSON.String()
		} else {
			reqBodyStr = string(bodyBytes)
		}

		// Let Go's http client handle Content-Length automatically
		// This is the most reliable way to ensure proper request transmission
		if len(bodyBytes) > 0 {
			req.ContentLength = int64(len(bodyBytes))
		} else {
			req.ContentLength = 0
		}
	}

	// Disable compression to make logs easier to read
	req.Header.Del("Accept-Encoding")

	// Log the full request
	t.logger.Debug("Outgoing request to backend",
		zap.String("backend", t.backend),
		zap.String("method", req.Method),
		zap.String("url", req.URL.String()),
		zap.Int64("content-length", req.ContentLength))

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
		// For streaming, use our enhanced DrainAndCapture function that maintains proper streaming
		resp.Body, respBodyStr = utils.DrainAndCapture(resp.Body, isStreaming)
	}

	// Log request and response details, being careful with streaming content
	if isStreaming {
		t.logger.Debug("Streaming response detected",
			zap.Int("status", resp.StatusCode),
			zap.String("contentType", resp.Header.Get("Content-Type")),
			zap.String("transferEncoding", resp.Header.Get("Transfer-Encoding")))

		// Log the headers separately
		for name, values := range resp.Header {
			t.logger.Debug("Response header",
				zap.String("name", name),
				zap.String("value", strings.Join(values, ", ")))
		}

		// Log a preview of the response content, even for streaming
		if len(respBodyStr) > 0 {
			t.logger.Debug("Streaming response preview", zap.String("content", respBodyStr))
		}
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

		// Store original request body if needed for modifications later
		var bodyBytes []byte
		if req.Body != nil && req.Method != "GET" {
			bodyBytes, _ = io.ReadAll(req.Body)
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// Set target host and URL
		req.Host = urlParsed.Host
		req.URL.Scheme = urlParsed.Scheme
		req.URL.Host = urlParsed.Host

		// Create proper path by joining base path with original path
		if strings.HasSuffix(urlParsed.Path, "/") && strings.HasPrefix(originalPath, "/") {
			// Avoid double slashes
			req.URL.Path = urlParsed.Path + originalPath[1:]
		} else if !strings.HasSuffix(urlParsed.Path, "/") && !strings.HasPrefix(originalPath, "/") {
			// Add slash when needed
			req.URL.Path = urlParsed.Path + "/" + originalPath
		} else {
			req.URL.Path = urlParsed.Path + originalPath
		}

		// Log the modifications to the request URL and Host
		logger.Info("Modified request URL and Host",
			zap.String("originalHost", originalHost),
			zap.String("newHost", req.Host),
			zap.String("originalPath", originalPath),
			zap.String("newPath", req.URL.Path),
		)

		// Extract client IP from RemoteAddr properly
		clientIP := req.RemoteAddr
		if idx := strings.LastIndex(clientIP, ":"); idx != -1 {
			clientIP = clientIP[:idx]
		}
		clientIP = strings.Trim(clientIP, "[]")

		// Set standard proxy headers
		// Note: We're setting these headers after clearing any existing ones to avoid duplication
		standardHeaders := map[string]string{
			"Host":              urlParsed.Host,
			"X-Real-IP":         clientIP,
			"X-Forwarded-Proto": "https",
			"X-Forwarded-Host":  originalHost,
		}

		for name, value := range standardHeaders {
			req.Header.Set(name, value)
		}

		// Handle X-Forwarded-For
		if xff := req.Header.Get("X-Forwarded-For"); xff != "" {
			req.Header.Set("X-Forwarded-For", fmt.Sprintf("%s, %s", xff, clientIP))
		} else {
			req.Header.Set("X-Forwarded-For", clientIP)
		}

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
