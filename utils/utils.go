package utils

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
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

	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		// If we can't read the body, return the original and an error message
		return body, fmt.Sprintf("Error reading body: %v", err)
	}

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
	// Add a flag to limit the captured size for very large responses
	maxCaptureSize int
	capturedSize   int
}

// NewResponseRecorder creates a new ResponseRecorder
func NewResponseRecorder(w http.ResponseWriter) *ResponseRecorder {
	return &ResponseRecorder{
		ResponseWriter: w,
		StatusCode:     http.StatusOK, // Default status code
		streaming:      false,
		maxCaptureSize: 1024 * 1024, // 1MB max capture size for logging (configurable)
		capturedSize:   0,
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
	// Write to the underlying ResponseWriter first
	n, err := r.ResponseWriter.Write(b)

	// If successful and we haven't exceeded our capture limit, record for logging
	if err == nil && n > 0 && r.capturedSize < r.maxCaptureSize {
		// Calculate how much we can safely add to our buffer
		remainingCapacity := r.maxCaptureSize - r.capturedSize
		if remainingCapacity > 0 {
			toCapture := b
			if len(b) > remainingCapacity {
				toCapture = b[:remainingCapacity]
			}

			bytesWritten, _ := r.Body.Write(toCapture)
			r.capturedSize += bytesWritten

			// If we hit the limit, add a note
			if r.capturedSize >= r.maxCaptureSize && len(b) > remainingCapacity {
				r.Body.WriteString("\n... [response truncated for logging, exceeded 1MB] ...")
			}
		}
	}

	return n, err
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

// GetBody returns the captured response body as a string, with special handling for SSE
func (r *ResponseRecorder) GetBody() string {
	if !r.streaming {
		return formatJSON(r.Body.Bytes())
	}

	// For streaming responses (SSE), try to reassemble the content in a more readable way
	content := r.Body.String()

	// Check if this is an SSE response with delta content (like OpenAI/Anthropic/etc)
	if strings.Contains(content, "data: {") && strings.Contains(content, "delta") {
		// Attempt to extract and combine content from stream
		var reassembledContent strings.Builder
		reassembledContent.WriteString("STREAMING RESPONSE (REASSEMBLED):\n")

		// Split by data: lines
		for _, line := range strings.Split(content, "data: ") {
			line = strings.TrimSpace(line)
			if line == "" || line == "[DONE]" {
				continue
			}

			// Extract content from the delta if possible
			if strings.HasPrefix(line, "{") {
				var jsonObj map[string]interface{}
				if err := json.Unmarshal([]byte(line), &jsonObj); err == nil {
					if choices, ok := jsonObj["choices"].([]interface{}); ok && len(choices) > 0 {
						if choice, ok := choices[0].(map[string]interface{}); ok {
							if delta, ok := choice["delta"].(map[string]interface{}); ok {
								if content, ok := delta["content"].(string); ok && content != "" {
									reassembledContent.WriteString(content)
								}
							}
						}
					}
				}
			}
		}

		// If we successfully extracted content, return it
		if reassembledContent.Len() > 25 { // More than just the header
			return reassembledContent.String()
		}
	}

	// If we couldn't reassemble or it's not delta content, return the raw content
	return "STREAMING CONTENT:\n" + content
}

// DrainAndCapture reads body content and returns it as both ReadCloser and string,
// but for streaming responses, samples more content without breaking the stream
func DrainAndCapture(body io.ReadCloser, isStreaming bool) (io.ReadCloser, string) {
	if body == nil {
		return nil, ""
	}

	// For streaming content, peek with a larger buffer
	if isStreaming {
		// Increase buffer size to 8KB to capture more of the stream
		peeked := make([]byte, 8*1024)
		n, err := body.Read(peeked)
		if err != nil && err != io.EOF {
			// If we can't read the body, return the original and an error message
			return body, fmt.Sprintf("Error peeking at streaming body: %v", err)
		}

		if n > 0 {
			peeked = peeked[:n]
			combinedReader := io.MultiReader(bytes.NewReader(peeked), body)

			// Try to parse and pretty-format the streamed content
			content := string(peeked)
			if strings.Contains(content, "data: {") && strings.Contains(content, "delta") {
				var builder strings.Builder
				builder.WriteString("STREAMING DATA (SAMPLE):\n")

				// Process the stream events we've captured so far
				for _, line := range strings.Split(content, "data: ") {
					line = strings.TrimSpace(line)
					if line == "" || line == "[DONE]" {
						continue
					}

					// Try to extract and format the delta content
					if strings.HasPrefix(line, "{") {
						var jsonObj map[string]interface{}
						if err := json.Unmarshal([]byte(line), &jsonObj); err == nil {
							prettyJSON, _ := json.MarshalIndent(jsonObj, "", "  ")
							builder.WriteString("--EVENT--\n")
							builder.Write(prettyJSON)
							builder.WriteString("\n")
						} else {
							builder.WriteString(line)
							builder.WriteString("\n")
						}
					}
				}

				return io.NopCloser(combinedReader), builder.String()
			}

			return io.NopCloser(combinedReader), "STREAMING: " + formatJSON(peeked) + "..."
		}
		return body, "STREAMING CONTENT (empty or could not be sampled)"
	}

	// For non-streaming content, buffer everything
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		// If we can't read the body, return the original and an error message
		return body, fmt.Sprintf("Error reading body: %v", err)
	}

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
