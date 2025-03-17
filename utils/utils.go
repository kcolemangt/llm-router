package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"unicode"

	"go.uber.org/zap"
)

// RedactAuthorization redacts sensitive information from authorization keys.
func RedactAuthorization(auth string) string {
	if strings.HasPrefix(auth, "Bearer ") && len(auth) > 29 {
		// Display the first 3 characters, ellipses, and the last 4 characters
		return auth[:10] + "..." + auth[len(auth)-4:]
	}
	// Replace each non-whitespace character with '*'
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return r
		}
		return '*'
	}, auth)
}

// DrainBody reads the body of an HTTP request and returns a new reader with the same content
// along with the body as a string for logging purposes.
func DrainBody(body io.ReadCloser) (io.ReadCloser, string) {
	if body == nil {
		return nil, ""
	}

	bodyBytes, _ := io.ReadAll(body)
	// Create new ReadClosers for the drained body
	return io.NopCloser(bytes.NewBuffer(bodyBytes)), formatJSON(bodyBytes)
}

// formatJSON attempts to format the byte array as JSON for better readability
// If it's not valid JSON, returns the string representation
func formatJSON(data []byte) string {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, data, "", "  "); err == nil {
		return prettyJSON.String()
	}
	return string(data)
}

// LogRequestResponse logs the full request and response details when debug logging is enabled
func LogRequestResponse(logger *zap.Logger, req *http.Request, resp *http.Response, reqBody, respBody string) {
	if req != nil {
		// Log request headers
		headers := make(map[string]string)
		for name, values := range req.Header {
			// Don't log authorization header directly
			if strings.ToLower(name) == "authorization" {
				headers[name] = RedactAuthorization(values[0])
			} else {
				headers[name] = strings.Join(values, ", ")
			}
		}

		logger.Debug("Full request details",
			zap.String("method", req.Method),
			zap.String("url", req.URL.String()),
			zap.Any("headers", headers),
			zap.String("body", reqBody),
		)
	}

	if resp != nil {
		// Log response headers
		headers := make(map[string]string)
		for name, values := range resp.Header {
			headers[name] = strings.Join(values, ", ")
		}

		logger.Debug("Full response details",
			zap.Int("status", resp.StatusCode),
			zap.Any("headers", headers),
			zap.String("body", respBody),
		)
	}
}

// ResponseRecorder is a custom http.ResponseWriter that captures the response body and status code
type ResponseRecorder struct {
	http.ResponseWriter
	StatusCode int
	Body       bytes.Buffer
	streaming  bool
}

// NewResponseRecorder creates a new ResponseRecorder
func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	return &ResponseRecorder{
		ResponseWriter: w,
		StatusCode:     http.StatusOK, // Default status code
		streaming:      false,
	}
}

// WriteHeader captures the status code and passes it to the underlying ResponseWriter
func (r *ResponseRecorder) WriteHeader(statusCode int) {
	r.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)

	// Check if this is a streaming response based on headers
	contentType := r.Header().Get("Content-Type")
	r.streaming = strings.Contains(contentType, "text/event-stream") || r.Header().Get("Transfer-Encoding") == "chunked"
}

// Write captures the response body and passes it to the underlying ResponseWriter
func (r *ResponseRecorder) Write(b []byte) (int, error) {
	// For streaming responses, just pass through without buffering everything
	if r.streaming {
		n, err := r.ResponseWriter.Write(b)
		if err == nil && n > 0 {
			// Log a sample of the response (first 100 chars max)
			sample := string(b)
			if len(sample) > 100 {
				sample = sample[:100] + "..."
			}
			r.Body.WriteString(sample)
		}
		return n, err
	}

	// For non-streaming responses, buffer the entire body
	r.Body.Write(b)
	return r.ResponseWriter.Write(b)
}

// Flush implements the http.Flusher interface to support streaming responses
func (r *ResponseRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Header returns the header map from the underlying ResponseWriter
func (r *ResponseRecorder) Header() http.Header {
	return r.ResponseWriter.Header()
}

// GetBody returns the captured response body as a string
func (r *ResponseRecorder) GetBody() string {
	return formatJSON(r.Body.Bytes())
}

// DrainAndCapture reads body content and returns it as both ReadCloser and string,
// but for streaming responses, only samples the beginning without reading everything
func DrainAndCapture(body io.ReadCloser, isStreaming bool) (io.ReadCloser, string) {
	if body == nil {
		return nil, ""
	}

	// For streaming content, just peek at the beginning
	if isStreaming {
		// Read only first 1KB to avoid breaking the stream
		peeked := make([]byte, 1024)
		n, _ := body.Read(peeked)
		if n > 0 {
			peeked = peeked[:n]
			combinedReader := io.MultiReader(bytes.NewReader(peeked), body)
			return io.NopCloser(combinedReader), "STREAMING: " + formatJSON(peeked) + "..."
		}
		return body, "STREAMING CONTENT"
	}

	// For non-streaming content, buffer everything
	bodyBytes, _ := io.ReadAll(body)
	return io.NopCloser(bytes.NewBuffer(bodyBytes)), formatJSON(bodyBytes)
}

// GenerateStrongAPIKey generates a cryptographically secure random API key
// with the format "rsk_" followed by random characters.
func GenerateStrongAPIKey() (string, error) {
	const keyLength = 48
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

	randomBytes := make([]byte, keyLength)
	_, err := io.ReadFull(rand.Reader, randomBytes)
	if err != nil {
		return "", err
	}

	result := make([]byte, keyLength)
	for i, b := range randomBytes {
		result[i] = charset[int(b)%len(charset)]
	}

	return "rsk_" + string(result), nil
}
