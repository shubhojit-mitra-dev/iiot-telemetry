package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var defaultDevices = []string{
	"CNC-001", "TURB-001", "ROBOT-001", "PUMP-001", "PRESS-001",
	"CONV-001", "KILN-001", "MILL-001", "AGV-001", "COMP-001",
}

func main() {
	targetURL := flag.String("target", "http://localhost:8080/api/v1/telemetry", "Target ingestion endpoint")
	totalMsgs := flag.Int("total", 50000, "Total number of messages to send")
	targetRate := flag.Int("rate", 10000, "Target messages/sec across all workers (0 for unlimited)")
	workers := flag.Int("workers", runtime.NumCPU()*4, "Number of concurrent worker goroutines")
	flag.Parse()

	fmt.Println("================================================================================")
	fmt.Println("           INDUSTRIAL IOT TELEMETRY ENGINE - STRESS BENCHMARK                   ")
	fmt.Println("================================================================================")
	fmt.Printf(" Target Endpoint    : %s\n", *targetURL)
	fmt.Printf(" Message Quota      : %d messages\n", *totalMsgs)
	if *targetRate > 0 {
		fmt.Printf(" Target Throughput  : %d msg/sec\n", *targetRate)
	} else {
		fmt.Println(" Target Throughput  : UNLIMITED (Max Hardware Saturation)")
	}
	fmt.Printf(" Concurrency Workers: %d\n", *workers)
	fmt.Printf(" Host Architecture  : %s / %s (%d CPU cores)\n", runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
	fmt.Println("--------------------------------------------------------------------------------")

	// High-throughput HTTP Transport optimized for zero-overhead socket pooling
	transport := &http.Transport{
		MaxIdleConns:        10000,
		MaxIdleConnsPerHost: 2000,
		MaxConnsPerHost:     0,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		DisableCompression:  true,
		ForceAttemptHTTP2:   false,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	// Warm up HTTP connection pool with single ping
	warmupPayload := fmt.Sprintf(`{"device_id":"CNC-001","timestamp":%d,"temperature":65.0,"vibration":0.5,"pressure":101.3}`, time.Now().UnixMilli())
	req, _ := http.NewRequest(http.MethodPost, *targetURL, bytes.NewBufferString(warmupPayload))
	req.Header.Set("Content-Type", "application/json")
	if resp, err := client.Do(req); err == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	} else {
		fmt.Printf("[ERROR] Target endpoint unreachable: %v\n", err)
		fmt.Println("Make sure the backend server is running on :8080 before starting benchmark.")
		return
	}

	fmt.Println(" Connection pool warmed up. Starting benchmark dispatch...")
	fmt.Println("--------------------------------------------------------------------------------")

	var (
		sentCount     int64
		acceptedCount int64
		droppedCount  int64
		errorCount    int64
	)

	// Latency sample collection (reservoir sampling up to 50,000 samples for percentile calculations)
	maxSamples := 50000
	latencies := make([]time.Duration, 0, maxSamples)
	var latMutex sync.Mutex

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	startTime := time.Now()

	// Rate limiter channel: provides tokens to control dispatch rate
	var tokenCh chan struct{}
	if *targetRate > 0 {
		tokenCh = make(chan struct{}, *targetRate*2)
		go func() {
			ticker := time.NewTicker(time.Second / time.Duration(*targetRate))
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					select {
					case tokenCh <- struct{}{}:
					default:
					}
				}
			}
		}()
	}

	// Live progress reporter
	doneChan := make(chan struct{})
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		lastSent := int64(0)
		lastTime := time.Now()

		for {
			select {
			case <-doneChan:
				return
			case <-ticker.C:
				now := time.Now()
				currSent := atomic.LoadInt64(&sentCount)
				deltaSent := currSent - lastSent
				elapsedSec := now.Sub(lastTime).Seconds()
				rate := float64(deltaSent) / elapsedSec

				currAccepted := atomic.LoadInt64(&acceptedCount)
				currDropped := atomic.LoadInt64(&droppedCount)
				currErr := atomic.LoadInt64(&errorCount)

				fmt.Printf(" [LIVE] Sent: %6d / %d | Speed: %7.0f msg/s | 202 Accepted: %6d | 429 Dropped: %d | Err: %d\n",
					currSent, *totalMsgs, rate, currAccepted, currDropped, currErr)

				lastSent = currSent
				lastTime = now
			}
		}
	}()

	var wg sync.WaitGroup
	chunkSize := *totalMsgs / *workers
	remainder := *totalMsgs % *workers

	for w := 0; w < *workers; w++ {
		workerLimit := chunkSize
		if w == 0 {
			workerLimit += remainder
		}

		wg.Add(1)
		go func(workerID, limit int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID*1000)))
			devCount := len(defaultDevices)

			// Pre-allocated payload buffer for zero alloc in worker loop
			buf := make([]byte, 256)

			for i := 0; i < limit; i++ {
				if tokenCh != nil {
					<-tokenCh
				}

				devID := defaultDevices[rng.Intn(devCount)]
				nowMs := time.Now().UnixMilli()
				temp := 60.0 + rng.Float64()*15.0
				vib := 0.2 + rng.Float64()*0.6
				press := 98.0 + rng.Float64()*6.0

				// Format fast JSON payload
				n := fmt.Sprintf(`{"device_id":"%s","timestamp":%d,"temperature":%.2f,"vibration":%.3f,"pressure":%.2f}`,
					devID, nowMs, temp, vib, press)
				copy(buf, n)

				reqStart := time.Now()
				httpReq, err := http.NewRequest(http.MethodPost, *targetURL, bytes.NewReader([]byte(n)))
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					atomic.AddInt64(&sentCount, 1)
					continue
				}
				httpReq.Header.Set("Content-Type", "application/json")

				resp, err := client.Do(httpReq)
				duration := time.Since(reqStart)

				atomic.AddInt64(&sentCount, 1)

				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					continue
				}

				_, _ = io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()

				if resp.StatusCode == http.StatusAccepted {
					atomic.AddInt64(&acceptedCount, 1)
				} else if resp.StatusCode == http.StatusTooManyRequests {
					atomic.AddInt64(&droppedCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}

				// Sample latency for P50 / P90 / P99
				latMutex.Lock()
				if len(latencies) < maxSamples {
					latencies = append(latencies, duration)
				} else if rng.Float64() < 0.1 {
					idx := rng.Intn(maxSamples)
					latencies[idx] = duration
				}
				latMutex.Unlock()
			}
		}(w, workerLimit)
	}

	wg.Wait()
	close(doneChan)
	totalDuration := time.Since(startTime)

	// Calculate latency percentiles
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	var p50, p90, p99, minLat, maxLat, avgLat time.Duration
	if len(latencies) > 0 {
		minLat = latencies[0]
		maxLat = latencies[len(latencies)-1]
		p50 = latencies[int(float64(len(latencies))*0.50)]
		p90 = latencies[int(float64(len(latencies))*0.90)]
		p99 = latencies[int(float64(len(latencies))*0.99)]

		var totalDurationSum time.Duration
		for _, lat := range latencies {
			totalDurationSum += lat
		}
		avgLat = totalDurationSum / time.Duration(len(latencies))
	}

	actualRate := float64(atomic.LoadInt64(&sentCount)) / totalDuration.Seconds()

	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("                             BENCHMARK RESULTS                                  ")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf(" Total Requests Sent: %d\n", atomic.LoadInt64(&sentCount))
	fmt.Printf(" HTTP 202 Accepted  : %d (%.2f%%)\n", atomic.LoadInt64(&acceptedCount), float64(atomic.LoadInt64(&acceptedCount))/float64(*totalMsgs)*100)
	fmt.Printf(" HTTP 429 Dropped   : %d (%.2f%%)\n", atomic.LoadInt64(&droppedCount), float64(atomic.LoadInt64(&droppedCount))/float64(*totalMsgs)*100)
	fmt.Printf(" Network / Err Drops: %d\n", atomic.LoadInt64(&errorCount))
	fmt.Printf(" Elapsed Time       : %v\n", totalDuration)
	fmt.Printf(" Realized Throughput: \033[1;32m%.1f msg/sec\033[0m\n", actualRate)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println("                        LATENCY PROFILES (P50 / P90 / P99)                      ")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf(" Minimum Latency    : %v\n", minLat)
	fmt.Printf(" Average Latency    : \033[1;33m%v\033[0m\n", avgLat)
	fmt.Printf(" P50 Median Latency : %v\n", p50)
	fmt.Printf(" P90 Latency        : %v\n", p90)
	fmt.Printf(" P99 Latency        : \033[1;36m%v\033[0m\n", p99)
	fmt.Printf(" Maximum Latency    : %v\n", maxLat)
	fmt.Println("================================================================================")
}
