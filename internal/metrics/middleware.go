package metrics

import (
	"net/http"
	"strings"
	"time"
)

// statusRecorder is a wrapper to capture status code
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// MetricsMiddleware wraps a handler to collect metrics
func MetricsMiddleware(collector *MetricsCollector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Extract apiKey from Authorization header
			auth := r.Header.Get("Authorization")
			apiKey := ""
			if strings.HasPrefix(auth, "Bearer ") {
				apiKey = strings.TrimPrefix(auth, "Bearer ")
			}

			// For now, hardcode model and tokens; integrate with tokenlimiter later
			model := "unknown" // TODO: Extract from request body or params
			tokens := 0        // TODO: Calculate from response or context

			endpoint := r.URL.Path

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			duration := time.Since(start)

			if apiKey != "" {
				collector.RecordRequest(apiKey, endpoint, model, tokens, rec.status, duration)
			}
		})
	}
}
