package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Yxp23/aegis/internal/api"
	"github.com/Yxp23/aegis/internal/providers/mock"
	"github.com/Yxp23/aegis/internal/router"
)

func percentile(values []time.Duration, p float64) time.Duration {
	index := int(float64(len(values)-1) * p)
	return values[index]
}

func main() {
	totalRequestsFlag := flag.Int(
		"requests",
		10000,
		"total number of requests",
	)

	concurrencyFlag := flag.Int(
		"concurrency",
		50,
		"number of concurrent workers",
	)

	flag.Parse()

	totalRequests := *totalRequestsFlag
	concurrency := *concurrencyFlag
	mockProvider := &mock.Provider{}
	r := router.New(mockProvider)

	server := httptest.NewServer(api.NewHandler(r))
	defer server.Close()

	transport := http.DefaultTransport.(*http.Transport).Clone()

	transport.MaxIdleConns = concurrency
	transport.MaxIdleConnsPerHost = concurrency
	transport.MaxConnsPerHost = concurrency

	client := &http.Client{
		Transport: transport,
	}

	defer transport.CloseIdleConnections()

	body := []byte(`{
		"model":"mock/mock-model",
		"messages":[
			{"role":"user","content":"load test"}
		]
	}`)

	jobs := make(chan struct{}, totalRequests)
	latencies := make([]time.Duration, totalRequests)

	var nextIndex atomic.Int64
	var failures atomic.Int64
	var wg sync.WaitGroup

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for range jobs {
				requestStart := time.Now()

				req, err := http.NewRequest(
					http.MethodPost,
					server.URL+"/v1/chat/completions",
					bytes.NewReader(body),
				)
				if err != nil {
					failures.Add(1)
					continue
				}

				req.Header.Set(
					"Content-Type",
					"application/json",
				)

				resp, err := client.Do(req)
				if err != nil {
					count := failures.Add(1)

					if count <= 5 {
						fmt.Printf(
							"request error: %v\n",
							err,
						)
					}

					continue
				}

				_, copyErr := io.Copy(io.Discard, resp.Body)
				resp.Body.Close()

				if copyErr != nil ||
					resp.StatusCode != http.StatusOK {
					failures.Add(1)
				}

				elapsed := time.Since(requestStart)

				index := nextIndex.Add(1) - 1
				latencies[index] = elapsed
			}
		}()
	}

	for i := 0; i < totalRequests; i++ {
		jobs <- struct{}{}
	}

	close(jobs)
	wg.Wait()

	totalTime := time.Since(start)

	completed := int(nextIndex.Load())
	latencies = latencies[:completed]

	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	requestsPerSecond :=
		float64(completed) / totalTime.Seconds()

	fmt.Printf("Requests:     %d\n", completed)
	fmt.Printf("Concurrency:  %d\n", concurrency)
	fmt.Printf("Failures:     %d\n", failures.Load())
	fmt.Printf("Duration:     %s\n", totalTime)
	fmt.Printf("Throughput:   %.2f req/s\n", requestsPerSecond)
	fmt.Printf("P50 latency:  %s\n", percentile(latencies, 0.50))
	fmt.Printf("P95 latency:  %s\n", percentile(latencies, 0.95))
	fmt.Printf("P99 latency:  %s\n", percentile(latencies, 0.99))
}
