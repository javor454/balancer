package server

import (
	"context"
)

// clientContextKey is used as a unique key for storing client information in context
// empty struct has 0 bytes size thus cheaper than string key, also prevents key key collission in context
type clientContextKey struct{}

// ClientContextKey is a singleton instance of clientContextKey used to store and retrieve client information
// from request context. Using a pointer to an empty struct ensures memory efficiency and thread safety.
var ClientContextKey = &clientContextKey{}

// ExtractClientFromContext extracts client name from request context
func ExtractClientFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	clientName, ok := ctx.Value(ClientContextKey).(string)
	if !ok {
		return ""
	}

	return clientName
}

// WithClientContext adds client information to the request context
func WithClientContext(ctx context.Context, clientName string) context.Context {
	return context.WithValue(ctx, ClientContextKey, clientName)
}
