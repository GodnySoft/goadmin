package web

import (
	"context"
	"math"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestWebLoadMetricsLatest100RPS проверяет, что endpoint /v1/metrics/latest
// выдерживает 100 RPS с приемлемой задержкой и долей ошибок.
func TestWebLoadMetricsLatest100RPS(t *testing.T) {
	t.Parallel()

	adapter := newTestAdapter(t, false, Config{})
	handler := adapter.routes()

	const (
		targetRPS = 100
		duration  = 10 * time.Second
	)
	totalRequests := int(duration.Seconds()) * targetRPS
	interval := time.Second / targetRPS

	type sample struct {
		latency time.Duration
		ok      bool
	}
	results := make(chan sample, totalRequests)

	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < totalRequests; i++ {
		scheduledAt := start.Add(time.Duration(i) * interval)
		sleep := time.Until(scheduledAt)
		if sleep > 0 {
			time.Sleep(sleep)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			reqStart := time.Now()
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/metrics/latest?module=host", nil)
			if err != nil {
				results <- sample{latency: time.Since(reqStart), ok: false}
				return
			}
			req.Header.Set("Authorization", "Bearer test-token")

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			resp := rr.Result()
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()

			results <- sample{latency: time.Since(reqStart), ok: resp.StatusCode == http.StatusOK}
		}()
	}

	wg.Wait()
	close(results)

	var (
		okCount   int64
		failCount int64
		latencies []time.Duration
	)
	for s := range results {
		if s.ok {
			atomic.AddInt64(&okCount, 1)
		} else {
			atomic.AddInt64(&failCount, 1)
		}
		latencies = append(latencies, s.latency)
	}

	if len(latencies) == 0 {
		t.Fatal("no latency samples collected")
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	p95 := percentile(latencies, 0.95)
	errorRate := float64(failCount) / float64(len(latencies))

	t.Logf("load summary: requests=%d ok=%d failed=%d error_rate=%.4f p95=%s", len(latencies), okCount, failCount, errorRate, p95)

	if p95 >= 250*time.Millisecond {
		t.Fatalf("p95 too high: got %s, want < 250ms", p95)
	}
	if errorRate > 0.01 {
		t.Fatalf("error rate too high: got %.4f, want <= 0.01", errorRate)
	}
}

func percentile(samples []time.Duration, p float64) time.Duration {
	if len(samples) == 0 {
		return 0
	}
	if p <= 0 {
		return samples[0]
	}
	if p >= 1 {
		return samples[len(samples)-1]
	}
	idx := int(math.Ceil(float64(len(samples))*p)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(samples) {
		idx = len(samples) - 1
	}
	return samples[idx]
}
