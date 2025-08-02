package metrics

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// BenchmarkMetricsCollectorRecordRequest benchmarks the core RecordRequest operation
func BenchmarkMetricsCollectorRecordRequest(b *testing.B) {
	collector := NewMetricsCollector()
	apiKey := "bench-key"
	endpoint := "/v1/test"
	model := "test-model"
	tokens := 100
	statusCode := 200
	duration := 100 * time.Millisecond

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		collector.RecordRequest(apiKey, endpoint, model, tokens, statusCode, duration)
	}
}

// BenchmarkMetricsCollectorRecordRequestParallel benchmarks concurrent RecordRequest calls
func BenchmarkMetricsCollectorRecordRequestParallel(b *testing.B) {
	collector := NewMetricsCollector()
	apiKey := "bench-key-parallel"
	endpoint := "/v1/test"
	model := "test-model"
	tokens := 100
	statusCode := 200
	duration := 100 * time.Millisecond

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			collector.RecordRequest(apiKey, endpoint, model, tokens, statusCode, duration)
		}
	})
}

// BenchmarkMetricsCollectorRecordRequestDifferentKeys benchmarks with different API keys
func BenchmarkMetricsCollectorRecordRequestDifferentKeys(b *testing.B) {
	collector := NewMetricsCollector()
	keys := make([]string, 100)
	for i := 0; i < 100; i++ {
		keys[i] = fmt.Sprintf("bench-key-%d", i)
	}

	endpoint := "/v1/test"
	model := "test-model"
	tokens := 100
	statusCode := 200
	duration := 100 * time.Millisecond

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		apiKey := keys[i%len(keys)]
		collector.RecordRequest(apiKey, endpoint, model, tokens, statusCode, duration)
	}
}

// BenchmarkMetricsCollectorRecordRequestDifferentEndpoints benchmarks with different endpoints
func BenchmarkMetricsCollectorRecordRequestDifferentEndpoints(b *testing.B) {
	collector := NewMetricsCollector()
	endpoints := []string{
		"/v1/chat/completions",
		"/v1/completions", 
		"/v1/embeddings",
		"/v1/images/generations",
		"/v1/models",
	}

	apiKey := "bench-key"
	model := "test-model"
	tokens := 100
	statusCode := 200
	duration := 100 * time.Millisecond

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		endpoint := endpoints[i%len(endpoints)]
		collector.RecordRequest(apiKey, endpoint, model, tokens, statusCode, duration)
	}
}

// BenchmarkMetricsCollectorGetMetrics benchmarks metrics retrieval
func BenchmarkMetricsCollectorGetMetrics(b *testing.B) {
	collector := NewMetricsCollector()
	
	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		apiKey := fmt.Sprintf("key-%d", i%100)
		endpoint := fmt.Sprintf("/v1/endpoint-%d", i%10)
		model := fmt.Sprintf("model-%d", i%5)
		collector.RecordRequest(apiKey, endpoint, model, 100, 200, 100*time.Millisecond)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		metrics := collector.GetMetrics()
		_ = metrics // Prevent optimization
	}
}

// BenchmarkMetricsCollectorGetMetricsParallel benchmarks concurrent metrics retrieval
func BenchmarkMetricsCollectorGetMetricsParallel(b *testing.B) {
	collector := NewMetricsCollector()
	
	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		apiKey := fmt.Sprintf("key-%d", i%100)
		endpoint := fmt.Sprintf("/v1/endpoint-%d", i%10)
		model := fmt.Sprintf("model-%d", i%5)
		collector.RecordRequest(apiKey, endpoint, model, 100, 200, 100*time.Millisecond)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			metrics := collector.GetMetrics()
			_ = metrics // Prevent optimization
		}
	})
}

// BenchmarkExportJSON benchmarks JSON export
func BenchmarkExportJSON(b *testing.B) {
	collector := NewMetricsCollector()
	
	// Pre-populate with realistic data
	for i := 0; i < 10000; i++ {
		apiKey := fmt.Sprintf("key-%d", i%500)
		endpoint := fmt.Sprintf("/v1/endpoint-%d", i%20)
		model := fmt.Sprintf("model-%d", i%10)
		tokens := 50 + rand.Intn(200)
		statusCode := 200
		if i%20 == 0 {
			statusCode = 500 // 5% error rate
		}
		collector.RecordRequest(apiKey, endpoint, model, tokens, statusCode, 100*time.Millisecond)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		data := ExportJSON(collector)
		_ = data // Prevent optimization
	}
}

// BenchmarkExportJSONParallel benchmarks concurrent JSON export
func BenchmarkExportJSONParallel(b *testing.B) {
	collector := NewMetricsCollector()
	
	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		apiKey := fmt.Sprintf("key-%d", i%100)
		endpoint := fmt.Sprintf("/v1/endpoint-%d", i%10)
		model := fmt.Sprintf("model-%d", i%5)
		collector.RecordRequest(apiKey, endpoint, model, 100, 200, 100*time.Millisecond)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			data := ExportJSON(collector)
			_ = data // Prevent optimization
		}
	})
}

// BenchmarkPrometheusHandler benchmarks Prometheus export
func BenchmarkPrometheusHandler(b *testing.B) {
	collector := NewMetricsCollector()
	
	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		apiKey := fmt.Sprintf("key-%d", i%100)
		endpoint := fmt.Sprintf("/v1/endpoint-%d", i%10)
		model := fmt.Sprintf("model-%d", i%5)
		collector.RecordRequest(apiKey, endpoint, model, 100, 200, 100*time.Millisecond)
	}

	handler := PrometheusHandler(collector)
	req := httptest.NewRequest("GET", "/metrics", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		_ = w.Body.String() // Ensure response is generated
	}
}

// BenchmarkMetricsMiddleware benchmarks the middleware overhead
func BenchmarkMetricsMiddleware(b *testing.B) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("POST", "/v1/test", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Authorization", "Bearer bench-key")
	req.Header.Set("Content-Type", "application/json")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}
}

// BenchmarkMetricsMiddlewareParallel benchmarks concurrent middleware usage
func BenchmarkMetricsMiddlewareParallel(b *testing.B) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	wrappedHandler := middleware(handler)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		req := httptest.NewRequest("POST", "/v1/test", strings.NewReader(`{"test":"data"}`))
		req.Header.Set("Authorization", "Bearer bench-key")
		req.Header.Set("Content-Type", "application/json")

		for pb.Next() {
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)
		}
	})
}

// BenchmarkMetricsMiddlewareWithDifferentKeys benchmarks middleware with different API keys
func BenchmarkMetricsMiddlewareWithDifferentKeys(b *testing.B) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)
	
	// Pre-create API keys to avoid allocation during benchmark
	keys := make([]string, 100)
	for i := 0; i < 100; i++ {
		keys[i] = fmt.Sprintf("bench-key-%d", i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		apiKey := keys[i%len(keys)]
		req := httptest.NewRequest("POST", "/v1/test", strings.NewReader(`{"test":"data"}`))
		req.Header.Set("Authorization", "Bearer "+apiKey)
		
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}
}

// BenchmarkMetricsMiddlewareWithDifferentEndpoints benchmarks middleware with different endpoints
func BenchmarkMetricsMiddlewareWithDifferentEndpoints(b *testing.B) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)
	
	endpoints := []string{
		"/v1/chat/completions",
		"/v1/completions",
		"/v1/embeddings", 
		"/v1/images/generations",
		"/v1/models",
		"/v1/fine-tunes",
		"/v1/files",
		"/health",
		"/metrics",
		"/status",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		endpoint := endpoints[i%len(endpoints)]
		req := httptest.NewRequest("POST", endpoint, strings.NewReader(`{"test":"data"}`))
		req.Header.Set("Authorization", "Bearer bench-key")
		
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}
}

// BenchmarkMemoryUsageScaling benchmarks memory usage scaling
func BenchmarkMemoryUsageScaling(b *testing.B) {
	scalingTests := []struct {
		name     string
		keys     int
		endpoints int
		models   int
	}{
		{"small_scale", 10, 5, 3},
		{"medium_scale", 100, 10, 5},
		{"large_scale", 1000, 20, 10},
		{"xlarge_scale", 10000, 50, 20},
	}

	for _, test := range scalingTests {
		b.Run(test.name, func(b *testing.B) {
			collector := NewMetricsCollector()
			
			// Pre-generate test data
			keys := make([]string, test.keys)
			endpoints := make([]string, test.endpoints)
			models := make([]string, test.models)
			
			for i := 0; i < test.keys; i++ {
				keys[i] = fmt.Sprintf("key-%d", i)
			}
			for i := 0; i < test.endpoints; i++ {
				endpoints[i] = fmt.Sprintf("/v1/endpoint-%d", i)
			}
			for i := 0; i < test.models; i++ {
				models[i] = fmt.Sprintf("model-%d", i)
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				apiKey := keys[i%len(keys)]
				endpoint := endpoints[i%len(endpoints)]
				model := models[i%len(models)]
				
				collector.RecordRequest(apiKey, endpoint, model, 100, 200, 100*time.Millisecond)
			}
		})
	}
}

// BenchmarkConcurrentReadWrite benchmarks concurrent read/write operations
func BenchmarkConcurrentReadWrite(b *testing.B) {
	collector := NewMetricsCollector()
	
	// Pre-populate with some data
	for i := 0; i < 100; i++ {
		collector.RecordRequest(fmt.Sprintf("key-%d", i), "/v1/test", "model", 100, 200, 100*time.Millisecond)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Simulate mixed read/write workload
			if rand.Intn(10) < 7 { // 70% writes, 30% reads
				apiKey := fmt.Sprintf("concurrent-key-%d", rand.Intn(100))
				endpoint := fmt.Sprintf("/v1/endpoint-%d", rand.Intn(10))
				model := fmt.Sprintf("model-%d", rand.Intn(5))
				collector.RecordRequest(apiKey, endpoint, model, 100, 200, 100*time.Millisecond)
			} else {
				metrics := collector.GetMetrics()
				_ = metrics // Prevent optimization
			}
		}
	})
}

// BenchmarkLatencyMeasurementOverhead benchmarks the overhead of latency measurement
func BenchmarkLatencyMeasurementOverhead(b *testing.B) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)

	// Handler with varying processing times
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate different processing times
		processingTime := time.Duration(rand.Intn(100)) * time.Microsecond
		time.Sleep(processingTime)
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("POST", "/v1/test", strings.NewReader(`{"test":"data"}`))
	req.Header.Set("Authorization", "Bearer bench-key")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}
}

// BenchmarkStatusRecorderOverhead benchmarks the overhead of the status recorder wrapper
func BenchmarkStatusRecorderOverhead(b *testing.B) {
	testCases := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{
			name: "simple_response",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
		},
		{
			name: "with_body",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("response body"))
			}),
		},
		{
			name: "with_headers",
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Custom-Header", "value")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"result":"success"}`))
			}),
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			collector := NewMetricsCollector()
			middleware := MetricsMiddleware(collector)
			wrappedHandler := middleware(tc.handler)

			req := httptest.NewRequest("POST", "/v1/test", nil)
			req.Header.Set("Authorization", "Bearer bench-key")

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				w := httptest.NewRecorder()
				wrappedHandler.ServeHTTP(w, req)
			}
		})
	}
}

// BenchmarkPrometheusCollectionOverhead benchmarks Prometheus collection overhead
func BenchmarkPrometheusCollectionOverhead(b *testing.B) {
	collector := NewMetricsCollector()
	
	// Pre-populate with data
	for i := 0; i < 1000; i++ {
		collector.RecordRequest(fmt.Sprintf("key-%d", i%100), "/v1/test", "model", 100, 200, 100*time.Millisecond)
	}

	// Test Prometheus collection directly
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		metricChan := make(chan prometheus.Metric, 100)
		go func() {
			defer close(metricChan)
			collector.Collect(metricChan)
		}()
		
		// Consume all metrics
		for range metricChan {
			// Process metric
		}
	}
}

// BenchmarkEndToEndLatency measures complete end-to-end latency
func BenchmarkEndToEndLatency(b *testing.B) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)

	// Realistic API handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate API processing time
		time.Sleep(50 * time.Microsecond)
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"result":"success","tokens":150}`))
	})

	wrappedHandler := middleware(handler)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v1/chat/completions", 
			strings.NewReader(`{"model":"gpt-4","messages":[{"role":"user","content":"test"}]}`))
		req.Header.Set("Authorization", "Bearer bench-key")
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		
		start := time.Now()
		wrappedHandler.ServeHTTP(w, req)
		latency := time.Since(start)
		
		// Verify the latency overhead is minimal
		if latency > 2*time.Millisecond { // Should be much less than 2ms overhead
			b.Errorf("Latency overhead too high: %v", latency)
		}
	}
}

// BenchmarkRealWorldScenario benchmarks a realistic API gateway scenario
func BenchmarkRealWorldScenario(b *testing.B) {
	collector := NewMetricsCollector()
	middleware := MetricsMiddleware(collector)

	// Realistic distribution of endpoints and response times
	endpoints := []struct {
		path     string
		latency  time.Duration
		tokens   int
		errRate  float64
	}{
		{"/v1/chat/completions", 250 * time.Millisecond, 150, 0.02},
		{"/v1/completions", 200 * time.Millisecond, 100, 0.01},
		{"/v1/embeddings", 100 * time.Millisecond, 50, 0.005},
		{"/health", 10 * time.Millisecond, 0, 0.0},
		{"/v1/models", 50 * time.Millisecond, 0, 0.0},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, ep := range endpoints {
			if r.URL.Path == ep.path {
				// Simulate processing time
				time.Sleep(ep.latency / 100) // Scale down for benchmark
				
				// Simulate error rate
				if rand.Float64() < ep.errRate {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				
				w.WriteHeader(http.StatusOK)
				if ep.tokens > 0 {
					w.Write([]byte(fmt.Sprintf(`{"usage":{"total_tokens":%d}}`, ep.tokens)))
				} else {
					w.Write([]byte(`{"status":"ok"}`))
				}
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	})

	wrappedHandler := middleware(handler)

	// Pre-generate requests for different API keys and endpoints
	requests := make([]*http.Request, 1000)
	for i := 0; i < 1000; i++ {
		endpoint := endpoints[rand.Intn(len(endpoints))]
		apiKey := fmt.Sprintf("api-key-%d", rand.Intn(50)) // 50 different API keys
		
		req := httptest.NewRequest("POST", endpoint.path, strings.NewReader(`{"test":"data"}`))
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Content-Type", "application/json")
		requests[i] = req
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := requests[i%len(requests)]
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}

	// Report some statistics
	metrics := collector.GetMetrics()
	b.Logf("Collected metrics for %d API keys", len(metrics))
	
	totalRequests := int64(0)
	for _, keyMetrics := range metrics {
		km := keyMetrics.(*KeyMetrics)
		totalRequests += km.TotalRequests
	}
	b.Logf("Total requests recorded: %d", totalRequests)
}