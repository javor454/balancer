package server

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"time"
)

type Middleware func(http.Handler) http.Handler

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

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func WithLogging(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			clientIP := r.Header.Get("X-Forwarded-For")
			if clientIP == "" {
				clientIP = r.RemoteAddr
			}

			requestBody, err := readBody(r)
			if err != nil {
				logger.Printf("Error reading request body: %v", err)
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

			logger.Printf(
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

func WithPanicRecovery(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Printf("Panic recovered: %v", err)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
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
