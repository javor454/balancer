package server

import (
	"fmt"
	"net/http"
)

func ValidateContentType(r *http.Request) error {
	if r.Header.Get("Content-Type") != "" && r.Header.Get("Content-Type") != "application/json" {
		return fmt.Errorf("unsupported media type")
	}

	return nil
}
