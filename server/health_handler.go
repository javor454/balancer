package server

import (
	"encoding/json"
	"net/http"
)

func healthHandler(proxyServerPool *ProxyServerPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.WriteHeader(http.StatusOK)
		response := map[string]any{
			"status":            "ok",
			"maxCapacity":       proxyServerPool.GetMaxCapacity(),
			"availableCapacity": proxyServerPool.GetAvailableCapacity(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}
}
