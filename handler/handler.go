package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/kcolemangt/llm-router/model"
	"github.com/kcolemangt/llm-router/proxy"
	"github.com/kcolemangt/llm-router/utils"
	"go.uber.org/zap"
)

// HandleRequest is the main HTTP handler function that processes incoming requests
func HandleRequest(cfg *model.Config, w http.ResponseWriter, r *http.Request) {
	// Check if this is likely a streaming request
	isStreaming := false
	if r.URL.Path == "/chat/completions" && r.Method == "POST" {
		contentType := r.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			// Read the first 1024 bytes to check for stream parameter
			// without consuming the entire body
			peeked := make([]byte, 1024)
			n, _ := r.Body.Read(peeked)
			if n > 0 {
				peeked = peeked[:n]
				isStreaming = strings.Contains(string(peeked), "\"stream\":true")
				// Restore the body
				combinedReader := io.MultiReader(bytes.NewReader(peeked), r.Body)
				r.Body = io.NopCloser(combinedReader)
			}
		}
	}

	// Create a response recorder to capture the response
	recorder := utils.NewResponseRecorder(w)

	// Log the full incoming request if debug is enabled
	var reqBody string
	if r.Body != nil {
		// For streaming, use a more careful approach to draining the body
		if isStreaming {
			r.Body, reqBody = utils.DrainAndCapture(r.Body, isStreaming)
		} else {
			r.Body, reqBody = utils.DrainBody(r.Body)
		}

		cfg.Logger.Debug("Incoming request",
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method),
			zap.Bool("streaming", isStreaming))
		utils.LogRequestResponse(cfg.Logger, r, nil, reqBody, "")
	}

	// Special handling for OPTIONS requests (CORS preflight)
	if r.Method == "OPTIONS" {
		cfg.Logger.Debug("Handling OPTIONS request for CORS preflight")

		// Get the request headers
		reqHeaders := r.Header.Get("Access-Control-Request-Headers")
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		// Log the request method requested in preflight
		if reqMethod := r.Header.Get("Access-Control-Request-Method"); reqMethod != "" {
			cfg.Logger.Debug("Preflight requested method", zap.String("method", reqMethod))
		}

		// Set CORS headers for OPTIONS requests
		recorder.Header().Set("Access-Control-Allow-Origin", origin)
		recorder.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")

		if reqHeaders != "" {
			recorder.Header().Set("Access-Control-Allow-Headers", reqHeaders)
		} else {
			recorder.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
		}

		recorder.Header().Set("Access-Control-Allow-Credentials", "true")
		recorder.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
		recorder.Header().Set("Content-Type", "text/plain")
		recorder.Header().Set("Content-Length", "0")
		recorder.Header().Set("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")

		// Return 204 No Content for OPTIONS requests
		recorder.WriteHeader(http.StatusNoContent)

		// Log the response
		logResponse(cfg.Logger, recorder)
		return
	}

	// Authenticate the request - only for non-OPTIONS requests
	authHeader := r.Header.Get("Authorization")
	expectedAuthHeader := "Bearer " + cfg.LLMRouterAPIKey
	if authHeader != expectedAuthHeader {
		cfg.Logger.Warn("Invalid or missing API key",
			zap.String("receivedAuthHeader", utils.RedactAuthorization(authHeader)),
			zap.String("expectedAuthHeader", utils.RedactAuthorization(expectedAuthHeader)))
		http.Error(recorder, "Invalid or missing API key", http.StatusUnauthorized)

		// Log the response
		logResponse(cfg.Logger, recorder)
		return
	}
	cfg.Logger.Info("API key validated successfully",
		zap.String("Authorization", utils.RedactAuthorization(authHeader)))

	// Process specific API endpoint logic if applicable
	if r.URL.Path == "/chat/completions" && r.Method == "POST" {
		handleChatCompletions(recorder, r, cfg.Logger)

		// Log the response
		logResponse(cfg.Logger, recorder)
		return
	}

	// Otherwise, route the request to the default backend
	routeRequestThroughProxy(r, recorder, cfg.Logger)

	// Log the response
	logResponse(cfg.Logger, recorder)
}

// logResponse logs the details of the HTTP response
func logResponse(logger *zap.Logger, recorder *utils.ResponseRecorder) {
	// Log response status and headers
	logger.Debug("Response details",
		zap.Int("status", recorder.StatusCode),
		zap.Any("headers", recorder.Header()),
		zap.String("body", recorder.GetBody()))
}

// handleChatCompletions processes specific logic for the chat completions endpoint
func handleChatCompletions(w http.ResponseWriter, r *http.Request, logger *zap.Logger) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var chatReq map[string]interface{}
	if err := json.Unmarshal(body, &chatReq); err != nil {
		http.Error(w, "Error unmarshalling request body", http.StatusInternalServerError)
		return
	}

	modelName, ok := chatReq["model"].(string)
	if !ok {
		http.Error(w, "Model key missing or not a string", http.StatusBadRequest)
		return
	}

	logger.Info("Incoming request for model", zap.String("model", modelName))

	for prefix, proxy := range proxy.Proxies {
		if strings.HasPrefix(modelName, prefix) {
			newModelName := strings.TrimPrefix(modelName, prefix)
			chatReq["model"] = newModelName
			modifiedBody, err := json.Marshal(chatReq)
			if err != nil {
				http.Error(w, "Error re-marshalling request body", http.StatusInternalServerError)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(modifiedBody))
			r.ContentLength = int64(len(modifiedBody))
			r.Header.Set("Content-Length", fmt.Sprintf("%d", len(modifiedBody)))

			logger.Info("Routing model to new model", zap.String("originalModel", modelName), zap.String("newModel", newModelName))

			proxy.ServeHTTP(w, r)
			return
		}
	}

	// If no prefix matches, use the default proxy
	if proxy.DefaultProxy != nil {
		logger.Info("Routing request to default proxy", zap.String("model", modelName))

		r.Body = io.NopCloser(bytes.NewBuffer(body))
		proxy.DefaultProxy.ServeHTTP(w, r)
		return
	}

	logger.Warn("No suitable backend found", zap.String("model", modelName))
	http.Error(w, "No suitable backend found", http.StatusBadGateway)
}

// routeRequestThroughProxy routes all generic requests through the default proxy
func routeRequestThroughProxy(r *http.Request, w http.ResponseWriter, logger *zap.Logger) {
	if proxy.DefaultProxy != nil {
		logger.Info("Routing request",
			zap.String("path", r.URL.Path),
			zap.String("method", r.Method))

		proxy.DefaultProxy.ServeHTTP(w, r)
	} else {
		logger.Info("No suitable backend configured for request",
			zap.String("path", r.URL.Path))
		http.Error(w, "No suitable backend configured", http.StatusBadGateway)
	}
}
