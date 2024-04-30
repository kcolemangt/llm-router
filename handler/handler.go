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
	// Authenticate the request
	authHeader := r.Header.Get("Authorization")
	expectedAuthHeader := "Bearer " + cfg.GlobalAPIKey
	if authHeader != expectedAuthHeader {
		cfg.Logger.Warn("Invalid or missing API key",
			zap.String("receivedAuthHeader", utils.RedactAuthorization(authHeader)),
			zap.String("expectedAuthHeader", utils.RedactAuthorization(expectedAuthHeader)))
		http.Error(w, "Invalid or missing API key", http.StatusUnauthorized)
		return
	}
	cfg.Logger.Info("API key validated successfully",
		zap.String("Authorization", utils.RedactAuthorization(authHeader)))

	// Process specific API endpoint logic if applicable
	if r.URL.Path == "/v1/chat/completions" && r.Method == "POST" {
		handleChatCompletions(w, r, cfg.Logger)
		return
	}

	// Otherwise, route the request to the default backend
	routeRequestThroughProxy(r, w, cfg.Logger)
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
		logger.Info("Routing general request",
			zap.String("path", r.URL.Path))
		proxy.DefaultProxy.ServeHTTP(w, r)
	} else {
		logger.Info("No suitable backend configured for request",
			zap.String("path", r.URL.Path))
		http.Error(w, "No suitable backend configured", http.StatusBadGateway)
	}
}
