package server

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"time"
)

type Middleware func(http.Handler) http.Handler

// Chain applies middlewares to a handler in reverse order e.g. A, B, C, H, C, B, A
func Chain(middlewares ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		handler := next
		for i := len(middlewares) - 1; i >= 0; i-- {
			handler = middlewares[i](handler)
		}
		return handler
	}
}

// WithLogging logs the request and response
func WithLogging() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			clientIP := r.Header.Get("X-Forwarded-For")
			if clientIP == "" {
				clientIP = r.RemoteAddr
			}

			requestBody, err := readBody(r)
			if err != nil {
				log.Printf("Error reading request body: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			wrapped := wrapResponseWriter(w)

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			params := make(map[string]string)
			if clientID := r.PathValue("clientID"); clientID != "" {
				params["clientID"] = clientID
			}

			sanitizedReqBody := sanitizeBody(requestBody)
			sanitizedResBody := sanitizeBody(wrapped.body.String())

			log.Printf(
				"Method: %s | Path: %s | IP: %s | Status: %d | Duration: %s | Params: %v | UserAgent: %s | "+
					"RequestBody: %s | ResponseBody: %s",
				r.Method,
				r.URL.Path,
				clientIP,
				wrapped.Status(),
				duration,
				params,
				r.UserAgent(),
				sanitizedReqBody,
				sanitizedResBody,
			)
		})
	}
}

// WithPanicRecovery recovers from panics and logs them
func WithPanicRecovery() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("Panic recovered: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// WithURLWhitelist creates a middleware that only allows requests to whitelisted URLs
func WithURLWhitelist(whitelist []string) Middleware {
	// Pre-compile the whitelist into a map for O(1) lookups
	whitelistMap := make(map[string]struct{}, len(whitelist))
	for _, path := range whitelist {
		whitelistMap[path] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if the requested path is in whitelist
			if _, allowed := whitelistMap[r.URL.Path]; !allowed {
				log.Printf("Blocked request to non-whitelisted URL: %s", r.URL.Path)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	body        *bytes.Buffer
}

func wrapResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		body:           &bytes.Buffer{},
	}
}

func (rw *responseWriter) Status() int {
	return rw.status
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.ResponseWriter.WriteHeader(code)
		rw.wroteHeader = true
	}
}

func readBody(r *http.Request) (string, error) {
	if r.Body == nil {
		return "", nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	r.Body = io.NopCloser(bytes.NewBuffer(body))

	return string(body), nil
}

func sanitizeBody(body string) string {
	maxLen := 1000
	if len(body) == 0 {
		return "empty"
	}

	if len(body) > maxLen {
		return body[:maxLen] + "..."
	}
	return body
}
