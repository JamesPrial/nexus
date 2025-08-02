package metrics

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMetricsMiddlewareRecordsMetrics(t *testing.T) {
	collector := NewMetricsCollector()
	mw := MetricsMiddleware(collector)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/v1/chat", nil)
	req.Header.Set("Authorization", "Bearer key1")
	// Assume token count is set somehow, e.g., in context or calculated

	rr := httptest.NewRecorder()
	mw(handler).ServeHTTP(rr, req)

	metrics := collector.GetMetrics()
	keyMetrics := metrics["key1"].(*KeyMetrics)

	assert.Equal(t, int64(1), keyMetrics.TotalRequests)
	assert.Equal(t, int64(1), keyMetrics.SuccessfulRequests)
	assert.Equal(t, 1, testutil.CollectAndCount(collector.RequestLatency)) // Check one observation
	// Assert tokens, endpoint, model - but need to set them in test
}

func TestMetricsMiddlewareCapturesFailure(t *testing.T) {
	collector := NewMetricsCollector()
	mw := MetricsMiddleware(collector)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})

	req := httptest.NewRequest("GET", "/v1/chat", nil)
	req.Header.Set("Authorization", "Bearer key1")

	rr := httptest.NewRecorder()
	mw(handler).ServeHTTP(rr, req)

	metrics := collector.GetMetrics()
	keyMetrics := metrics["key1"].(*KeyMetrics)

	assert.Equal(t, int64(1), keyMetrics.TotalRequests)
	assert.Equal(t, int64(0), keyMetrics.SuccessfulRequests)
	assert.Equal(t, int64(1), keyMetrics.FailedRequests)
}

func TestMetricsMiddlewareConcurrentRequests(t *testing.T) {
	collector := NewMetricsCollector()
	mw := MetricsMiddleware(collector)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/v1/chat", nil)
			req.Header.Set("Authorization", "Bearer key1")
			rr := httptest.NewRecorder()
			mw(handler).ServeHTTP(rr, req)
		}()
	}
	wg.Wait()

	metrics := collector.GetMetrics()
	keyMetrics := metrics["key1"].(*KeyMetrics)

	assert.Equal(t, int64(50), keyMetrics.TotalRequests)
	assert.Equal(t, int64(50), keyMetrics.SuccessfulRequests)
}
