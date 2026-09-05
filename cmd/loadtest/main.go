package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

func percentile(values []time.Duration, p float64) time.Duration {
	if len(values) == 0 {
		return 0
	}

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

	targetURLFlag := flag.String(
		"url",
		"http://localhost:9090",
		"target Aegis server URL",
	)

	flag.Parse()

	totalRequests := *totalRequestsFlag
	concurrency := *concurrencyFlag
	targetURL := *targetURLFlag

	if totalRequests <= 0 {
		fmt.Println("requests must be greater than 0")
		return
	}

	if concurrency <= 0 {
		fmt.Println("concurrency must be greater than 0")
		return
	}

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

	latencies := make(
		[]time.Duration,
		totalRequests,
	)

	var completed atomic.Int64
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
					targetURL+"/v1/chat/completions",
					bytes.NewReader(body),
				)
				if err != nil {
					count := failures.Add(1)

					if count <= 5 {
						fmt.Printf(
							"request creation error: %v\n",
							err,
						)
					}

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

				_, copyErr := io.Copy(
					io.Discard,
					resp.Body,
				)

				resp.Body.Close()

				if copyErr != nil {
					count := failures.Add(1)

					if count <= 5 {
						fmt.Printf(
							"response read error: %v\n",
							copyErr,
						)
					}

					continue
				}

				if resp.StatusCode != http.StatusOK {
					count := failures.Add(1)

					if count <= 5 {
						fmt.Printf(
							"unexpected status: %d\n",
							resp.StatusCode,
						)
					}

					continue
				}

				elapsed := time.Since(requestStart)

				index := completed.Add(1) - 1
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

	completedRequests := int(completed.Load())
	failedRequests := failures.Load()

	latencies = latencies[:completedRequests]

	sort.Slice(
		latencies,
		func(i, j int) bool {
			return latencies[i] < latencies[j]
		},
	)

	requestsPerSecond := 0.0

	if totalTime > 0 {
		requestsPerSecond =
			float64(completedRequests) /
				totalTime.Seconds()
	}

	fmt.Printf(
		"Target:       %s\n",
		targetURL,
	)
	fmt.Printf(
		"Attempted:    %d\n",
		totalRequests,
	)
	fmt.Printf(
		"Completed:    %d\n",
		completedRequests,
	)
	fmt.Printf(
		"Concurrency:  %d\n",
		concurrency,
	)
	fmt.Printf(
		"Failures:     %d\n",
		failedRequests,
	)
	fmt.Printf(
		"Duration:     %s\n",
		totalTime,
	)
	fmt.Printf(
		"Throughput:   %.2f req/s\n",
		requestsPerSecond,
	)

	if completedRequests > 0 {
		fmt.Printf(
			"P50 latency:  %s\n",
			percentile(latencies, 0.50),
		)
		fmt.Printf(
			"P95 latency:  %s\n",
			percentile(latencies, 0.95),
		)
		fmt.Printf(
			"P99 latency:  %s\n",
			percentile(latencies, 0.99),
		)
	}
}
