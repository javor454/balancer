package server

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/javor454/balancer/auth"
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
			sanitizedResBody := sanitizeBody(wrapped.body.String()) // why string conversion

			log.Printf(
				"Method: %s | Path: %s | IP: %s | Status: %d | Duration: %s | Params: %v | UserAgent: %s | RequestBody: %s | ResponseBody: %s",
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
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				defer func() {
					if err := recover(); err != nil {
						log.Printf("Panic recovered: %v", err)
						http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					}
				}()
				next.ServeHTTP(w, r)
			},
		)
	}
}

// WithWhitelistedPaths allows requests only to whitelisted paths
func WithWhitelistedPaths(whitelist []string) Middleware {
	whitelistedPathsLookup := make(map[string]struct{}, len(whitelist))
	for _, path := range whitelist {
		whitelistedPathsLookup[path] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				if _, allowed := whitelistedPathsLookup[r.URL.Path]; !allowed {
					log.Printf("Blocked request to non-whitelisted path: %s", r.URL.Path)
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
			},
		)
	}
}

// WithConditionalAuth checks authorization header only to paths that are not in the blacklist
func WithConditionalAuth(blacklistedPaths []string, authHandler *auth.AuthHandler) Middleware {
	blacklistedPathsLookup := make(map[string]struct{})
	for _, path := range blacklistedPaths {
		blacklistedPathsLookup[path] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				// Skip auth for excluded paths
				if _, isExcluded := blacklistedPathsLookup[r.URL.Path]; isExcluded {
					next.ServeHTTP(w, r)
					return
				}

				if r.Header.Get("Authorization") == "" {
					log.Printf("Empty authorization header for path: %s", r.URL.Path)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				if !authHandler.VerifyRegistered(r.Header.Get("Authorization")) {
					log.Printf("Unauthorized request to path: %s", r.URL.Path)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				next.ServeHTTP(w, r)
			},
		)
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode  int
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
	return rw.statusCode
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
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


// sanitizeBody shortens the body to 1000 characters
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
