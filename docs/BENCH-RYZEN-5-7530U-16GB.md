# Benchmark Report: AMD Ryzen 5 7530U (16GB RAM)

- **Target Machine:** AMD Ryzen 5 7530U with Radeon Graphics (6 Cores, 12 Logical Processors)
- **RAM:** 16.00 GB DDR4
- **Operating System:** Windows 11 (windows / amd64)
- **Engine:** Go 1.27 runtime (`sync.Pool` zero-allocation telemetry pipeline)
- **Date:** October 1, 2026

---

## 1. Executive Summary & Key Discoveries

### Peak Performance Realized (Unthrottled Saturation)
- **Peak Throughput:** **48,743.1 msgs/sec**
- **Payload Volume:** 200,000 messages ingested in **4.10 seconds**
- **Success Rate:** **100.00% (200,000 / 200,000 HTTP 202 Accepted)**
- **Dropped Messages:** **0 (0.00%)**
- **Network / Socket Errors:** **0**
- **Average Latency:** **1.16 ms**
- **P50 Median Latency:** **1.01 ms**
- **P90 Latency:** **2.30 ms**
- **P99 Latency:** **4.77 ms**
- **Maximum Latency:** **17.08 ms**

---

## 2. Technical Analysis: Why the Numbers Looked Confusing

During the sequential benchmark runs, three surprising behaviors were observed:
1. Why did the 1,000 msg/sec run hit 999.3 msg/sec, but the 5k, 10k, 25k, and 50k runs all plateaued at **~1,820 – 1,840 msg/sec**?
2. Why did the latency for the rate-limited runs show **0s, 647ns, and 1.4µs**? Was it the CPU cache?
3. Why did the unthrottled run (`-rate=0`) suddenly leap to **48,743 msg/sec** with real millisecond latencies (~1.16ms)?

### Phenomenon A: The ~1,830 msg/sec Rate-Limiting Plateau
- In `cmd/bench/main.go`, the target rate was controlled by a token channel driven by `time.NewTicker(time.Second / targetRate)`.
- For `-rate=1000`, the interval is `1ms`, which Windows handles comfortably.
- For `-rate=10000`, the interval is `100µs`. For `-rate=50000`, the interval is `20µs`.
- **The Windows OS Timer Resolution Limitation:** On Windows, the OS thread scheduler and system clock quantum cannot wake a single user-space goroutine every 20 microseconds. Standard Windows timer interrupts cap single-threaded ticker loops at roughly **1,800 to 1,850 ticks per second** (~0.54ms per tick).
- **Conclusion:** The backend server was never bottlenecked. The benchmarking client itself was bottlenecked by the Windows kernel timer resolution, feeding only ~1,830 tokens per second to the workers.

### Phenomenon B: The "Nanosecond" Latency Myth (Windows Clock Quantization)
- In the rate-limited runs, workers spent almost all their time waiting on `<-tokenCh`. When a token was released, a single request was sent over loopback TCP keep-alive and completed in ~0.3ms to 0.4ms.
- **Clock Quantization:** In Windows, system time is updated in discrete quanta (often ~1ms). If `time.Since(reqStart)` is evaluated before the OS advances the system clock tick, Go records a duration of `0s`.
- When 99%+ of samples finish within the same clock tick, the sorted array contains `0s` at the P50, P90, and P99 positions. Dividing the few samples that crossed tick boundaries across 50,000 requests mathematically produced an artificial average of **647ns to 1.4µs**.
- **Conclusion:** This was not CPU L3 cache or Redis cache; it was OS timer quantization.

### Phenomenon C: The Unthrottled Saturation Test (`-rate=0`)
- When running `-rate=0`, the ticker was completely bypassed (`tokenCh == nil`).
- All 64 workers fired HTTP requests concurrently across the 12 CPU threads without sleeping.
- The server absorbed **48,743 requests per second**, processing all 200,000 requests in **4.10 seconds**.
- Under this full pipeline saturation, real network buffer transit times crossed the timer quantum, measuring accurate millisecond metrics: **1.01ms P50, 2.30ms P90, and 4.77ms P99**, with **zero drops**.

---

## 3. Raw Benchmark Execution Transcript (Unedited)

```text
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry> go run ./cmd/bench/main.go -rate=1000 -total=10000 -workers=8
================================================================================
           INDUSTRIAL IOT TELEMETRY ENGINE - STRESS BENCHMARK
================================================================================
 Target Endpoint    : http://localhost:8080/api/v1/telemetry
 Message Quota      : 10000 messages
 Target Throughput  : 1000 msg/sec
 Concurrency Workers: 8
 Host Architecture  : windows / amd64 (12 CPU cores)
--------------------------------------------------------------------------------
 Connection pool warmed up. Starting benchmark dispatch...
--------------------------------------------------------------------------------
 [LIVE] Sent:    997 / 10000 | Speed:     997 msg/s | 202 Accepted:    997 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   1997 / 10000 | Speed:    1000 msg/s | 202 Accepted:   1997 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   2995 / 10000 | Speed:     998 msg/s | 202 Accepted:   2995 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   3995 / 10000 | Speed:    1000 msg/s | 202 Accepted:   3995 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   4993 / 10000 | Speed:     998 msg/s | 202 Accepted:   4993 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   5993 / 10000 | Speed:    1000 msg/s | 202 Accepted:   5993 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   6992 / 10000 | Speed:     999 msg/s | 202 Accepted:   6992 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   7992 / 10000 | Speed:    1000 msg/s | 202 Accepted:   7992 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   8992 / 10000 | Speed:    1000 msg/s | 202 Accepted:   8992 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   9992 / 10000 | Speed:    1000 msg/s | 202 Accepted:   9992 | 429 Dropped: 0 | Err: 0
--------------------------------------------------------------------------------
                             BENCHMARK RESULTS
--------------------------------------------------------------------------------
 Total Requests Sent: 10000
 HTTP 202 Accepted  : 10000 (100.00%)
 HTTP 429 Dropped   : 0 (0.00%)
 Network / Err Drops: 0
 Elapsed Time       : 10.0073228s
 Realized Throughput: 999.3 msg/sec
--------------------------------------------------------------------------------
                        LATENCY PROFILES (P50 / P90 / P99)
--------------------------------------------------------------------------------
 Minimum Latency    : 0s
 Average Latency    : 1.487µs
 P50 Median Latency : 0s
 P90 Latency        : 0s
 P99 Latency        : 0s
 Maximum Latency    : 1.9875ms
================================================================================
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry> go run ./cmd/bench/main.go -rate=5000 -total=25000 -workers=16
================================================================================
           INDUSTRIAL IOT TELEMETRY ENGINE - STRESS BENCHMARK
================================================================================
 Target Endpoint    : http://localhost:8080/api/v1/telemetry
 Message Quota      : 25000 messages
 Target Throughput  : 5000 msg/sec
 Concurrency Workers: 16
 Host Architecture  : windows / amd64 (12 CPU cores)
--------------------------------------------------------------------------------
 Connection pool warmed up. Starting benchmark dispatch...
--------------------------------------------------------------------------------
 [LIVE] Sent:   1818 / 25000 | Speed:    1818 msg/s | 202 Accepted:   1818 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   3632 / 25000 | Speed:    1814 msg/s | 202 Accepted:   3632 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   5446 / 25000 | Speed:    1813 msg/s | 202 Accepted:   5446 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   7264 / 25000 | Speed:    1818 msg/s | 202 Accepted:   7264 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   9090 / 25000 | Speed:    1826 msg/s | 202 Accepted:   9090 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  10913 / 25000 | Speed:    1823 msg/s | 202 Accepted:  10913 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  12732 / 25000 | Speed:    1819 msg/s | 202 Accepted:  12732 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  14558 / 25000 | Speed:    1826 msg/s | 202 Accepted:  14558 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  16375 / 25000 | Speed:    1816 msg/s | 202 Accepted:  16375 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  18205 / 25000 | Speed:    1830 msg/s | 202 Accepted:  18205 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  20046 / 25000 | Speed:    1841 msg/s | 202 Accepted:  20046 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  21882 / 25000 | Speed:    1836 msg/s | 202 Accepted:  21882 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  23718 / 25000 | Speed:    1836 msg/s | 202 Accepted:  23718 | 429 Dropped: 0 | Err: 0
--------------------------------------------------------------------------------
                             BENCHMARK RESULTS
--------------------------------------------------------------------------------
 Total Requests Sent: 25000
 HTTP 202 Accepted  : 25000 (100.00%)
 HTTP 429 Dropped   : 0 (0.00%)
 Network / Err Drops: 0
 Elapsed Time       : 13.7008383s
 Realized Throughput: 1824.7 msg/sec
--------------------------------------------------------------------------------
                        LATENCY PROFILES (P50 / P90 / P99)
--------------------------------------------------------------------------------
 Minimum Latency    : 0s
 Average Latency    : 1.115µs
 P50 Median Latency : 0s
 P90 Latency        : 0s
 P99 Latency        : 0s
 Maximum Latency    : 1.0564ms
================================================================================
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry> go run ./cmd/bench/main.go -rate=10000 -total=50000 -workers=32
================================================================================
           INDUSTRIAL IOT TELEMETRY ENGINE - STRESS BENCHMARK
================================================================================
 Target Endpoint    : http://localhost:8080/api/v1/telemetry
 Message Quota      : 50000 messages
 Target Throughput  : 10000 msg/sec
 Concurrency Workers: 32
 Host Architecture  : windows / amd64 (12 CPU cores)
--------------------------------------------------------------------------------
 Connection pool warmed up. Starting benchmark dispatch...
--------------------------------------------------------------------------------
 [LIVE] Sent:   1836 / 50000 | Speed:    1835 msg/s | 202 Accepted:   1836 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   3671 / 50000 | Speed:    1835 msg/s | 202 Accepted:   3671 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   5505 / 50000 | Speed:    1835 msg/s | 202 Accepted:   5505 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   7342 / 50000 | Speed:    1836 msg/s | 202 Accepted:   7342 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   9177 / 50000 | Speed:    1835 msg/s | 202 Accepted:   9177 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  11018 / 50000 | Speed:    1841 msg/s | 202 Accepted:  11018 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  12858 / 50000 | Speed:    1840 msg/s | 202 Accepted:  12858 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  14698 / 50000 | Speed:    1839 msg/s | 202 Accepted:  14698 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  16535 / 50000 | Speed:    1837 msg/s | 202 Accepted:  16535 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  18375 / 50000 | Speed:    1840 msg/s | 202 Accepted:  18375 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  20203 / 50000 | Speed:    1828 msg/s | 202 Accepted:  20203 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  22049 / 50000 | Speed:    1846 msg/s | 202 Accepted:  22049 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  23891 / 50000 | Speed:    1842 msg/s | 202 Accepted:  23891 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  25734 / 50000 | Speed:    1843 msg/s | 202 Accepted:  25734 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  27573 / 50000 | Speed:    1839 msg/s | 202 Accepted:  27573 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  29413 / 50000 | Speed:    1840 msg/s | 202 Accepted:  29413 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  31253 / 50000 | Speed:    1840 msg/s | 202 Accepted:  31253 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  33090 / 50000 | Speed:    1837 msg/s | 202 Accepted:  33090 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  34925 / 50000 | Speed:    1834 msg/s | 202 Accepted:  34925 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  36757 / 50000 | Speed:    1833 msg/s | 202 Accepted:  36757 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  38597 / 50000 | Speed:    1840 msg/s | 202 Accepted:  38597 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  40437 / 50000 | Speed:    1840 msg/s | 202 Accepted:  40437 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  42271 / 50000 | Speed:    1834 msg/s | 202 Accepted:  42271 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  44110 / 50000 | Speed:    1839 msg/s | 202 Accepted:  44110 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  45943 / 50000 | Speed:    1834 msg/s | 202 Accepted:  45943 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  47778 / 50000 | Speed:    1834 msg/s | 202 Accepted:  47778 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  49602 / 50000 | Speed:    1825 msg/s | 202 Accepted:  49602 | 429 Dropped: 0 | Err: 0
--------------------------------------------------------------------------------
                             BENCHMARK RESULTS
--------------------------------------------------------------------------------
 Total Requests Sent: 50000
 HTTP 202 Accepted  : 50000 (100.00%)
 HTTP 429 Dropped   : 0 (0.00%)
 Network / Err Drops: 0
 Elapsed Time       : 27.2176916s
 Realized Throughput: 1837.0 msg/sec
--------------------------------------------------------------------------------
                        LATENCY PROFILES (P50 / P90 / P99)
--------------------------------------------------------------------------------
 Minimum Latency    : 0s
 Average Latency    : 662ns
 P50 Median Latency : 0s
 P90 Latency        : 0s
 P99 Latency        : 0s
 Maximum Latency    : 1.1334ms
================================================================================
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry> go run ./cmd/bench/main.go -rate=25000 -total=100000 -workers=48
================================================================================
           INDUSTRIAL IOT TELEMETRY ENGINE - STRESS BENCHMARK
================================================================================
 Target Endpoint    : http://localhost:8080/api/v1/telemetry
 Message Quota      : 100000 messages
 Target Throughput  : 25000 msg/sec
 Concurrency Workers: 48
 Host Architecture  : windows / amd64 (12 CPU cores)
--------------------------------------------------------------------------------
 Connection pool warmed up. Starting benchmark dispatch...
--------------------------------------------------------------------------------
 [LIVE] Sent:   1840 / 100000 | Speed:    1839 msg/s | 202 Accepted:   1840 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   3680 / 100000 | Speed:    1840 msg/s | 202 Accepted:   3680 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   5526 / 100000 | Speed:    1846 msg/s | 202 Accepted:   5526 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   7379 / 100000 | Speed:    1852 msg/s | 202 Accepted:   7379 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   9223 / 100000 | Speed:    1844 msg/s | 202 Accepted:   9223 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  11073 / 100000 | Speed:    1851 msg/s | 202 Accepted:  11073 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  12921 / 100000 | Speed:    1848 msg/s | 202 Accepted:  12921 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  14759 / 100000 | Speed:    1838 msg/s | 202 Accepted:  14759 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  16608 / 100000 | Speed:    1849 msg/s | 202 Accepted:  16608 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  18459 / 100000 | Speed:    1851 msg/s | 202 Accepted:  18459 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  20313 / 100000 | Speed:    1854 msg/s | 202 Accepted:  20313 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  22164 / 100000 | Speed:    1851 msg/s | 202 Accepted:  22164 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  24011 / 100000 | Speed:    1847 msg/s | 202 Accepted:  24011 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  25853 / 100000 | Speed:    1842 msg/s | 202 Accepted:  25853 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  27700 / 100000 | Speed:    1847 msg/s | 202 Accepted:  27700 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  29536 / 100000 | Speed:    1836 msg/s | 202 Accepted:  29536 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  31381 / 100000 | Speed:    1845 msg/s | 202 Accepted:  31381 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  33223 / 100000 | Speed:    1842 msg/s | 202 Accepted:  33223 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  35071 / 100000 | Speed:    1848 msg/s | 202 Accepted:  35071 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  36917 / 100000 | Speed:    1846 msg/s | 202 Accepted:  36917 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  38762 / 100000 | Speed:    1845 msg/s | 202 Accepted:  38762 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  40601 / 100000 | Speed:    1839 msg/s | 202 Accepted:  40601 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  42441 / 100000 | Speed:    1840 msg/s | 202 Accepted:  42441 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  44289 / 100000 | Speed:    1848 msg/s | 202 Accepted:  44289 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  46135 / 100000 | Speed:    1846 msg/s | 202 Accepted:  46135 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  47972 / 100000 | Speed:    1837 msg/s | 202 Accepted:  47972 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  49817 / 100000 | Speed:    1845 msg/s | 202 Accepted:  49817 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  51655 / 100000 | Speed:    1838 msg/s | 202 Accepted:  51655 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  53498 / 100000 | Speed:    1842 msg/s | 202 Accepted:  53498 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  55335 / 100000 | Speed:    1838 msg/s | 202 Accepted:  55335 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  57178 / 100000 | Speed:    1843 msg/s | 202 Accepted:  57178 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  59024 / 100000 | Speed:    1846 msg/s | 202 Accepted:  59024 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  60867 / 100000 | Speed:    1843 msg/s | 202 Accepted:  60867 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  62713 / 100000 | Speed:    1846 msg/s | 202 Accepted:  62713 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  64564 / 100000 | Speed:    1851 msg/s | 202 Accepted:  64564 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  66413 / 100000 | Speed:    1849 msg/s | 202 Accepted:  66413 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  68261 / 100000 | Speed:    1848 msg/s | 202 Accepted:  68261 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  70108 / 100000 | Speed:    1847 msg/s | 202 Accepted:  70108 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  71959 / 100000 | Speed:    1851 msg/s | 202 Accepted:  71959 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  73798 / 100000 | Speed:    1838 msg/s | 202 Accepted:  73798 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  75650 / 100000 | Speed:    1853 msg/s | 202 Accepted:  75650 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  77499 / 100000 | Speed:    1849 msg/s | 202 Accepted:  77499 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  79343 / 100000 | Speed:    1844 msg/s | 202 Accepted:  79343 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  81183 / 100000 | Speed:    1840 msg/s | 202 Accepted:  81183 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  83027 / 100000 | Speed:    1844 msg/s | 202 Accepted:  83027 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  84863 / 100000 | Speed:    1835 msg/s | 202 Accepted:  84863 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  86704 / 100000 | Speed:    1841 msg/s | 202 Accepted:  86704 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  88549 / 100000 | Speed:    1845 msg/s | 202 Accepted:  88549 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  90387 / 100000 | Speed:    1838 msg/s | 202 Accepted:  90387 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  92232 / 100000 | Speed:    1845 msg/s | 202 Accepted:  92232 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  94072 / 100000 | Speed:    1840 msg/s | 202 Accepted:  94072 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  95916 / 100000 | Speed:    1844 msg/s | 202 Accepted:  95916 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  97752 / 100000 | Speed:    1836 msg/s | 202 Accepted:  97752 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  99588 / 100000 | Speed:    1836 msg/s | 202 Accepted:  99588 | 429 Dropped: 0 | Err: 0
--------------------------------------------------------------------------------
                             BENCHMARK RESULTS
--------------------------------------------------------------------------------
 Total Requests Sent: 100000
 HTTP 202 Accepted  : 100000 (100.00%)
 HTTP 429 Dropped   : 0 (0.00%)
 Network / Err Drops: 0
 Elapsed Time       : 54.2269877s
 Realized Throughput: 1844.1 msg/sec
--------------------------------------------------------------------------------
                        LATENCY PROFILES (P50 / P90 / P99)
--------------------------------------------------------------------------------
 Minimum Latency    : 0s
 Average Latency    : 1.049µs
 P50 Median Latency : 0s
 P90 Latency        : 0s
 P99 Latency        : 0s
 Maximum Latency    : 3.3985ms
================================================================================
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry> go run ./cmd/bench/main.go -rate=25000 -total=100000 -workers=48
================================================================================
           INDUSTRIAL IOT TELEMETRY ENGINE - STRESS BENCHMARK
================================================================================
 Target Endpoint    : http://localhost:8080/api/v1/telemetry
 Message Quota      : 100000 messages
 Target Throughput  : 25000 msg/sec
 Concurrency Workers: 48
 Host Architecture  : windows / amd64 (12 CPU cores)
--------------------------------------------------------------------------------
 Connection pool warmed up. Starting benchmark dispatch...
--------------------------------------------------------------------------------
 [LIVE] Sent:   1837 / 100000 | Speed:    1836 msg/s | 202 Accepted:   1837 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   3667 / 100000 | Speed:    1830 msg/s | 202 Accepted:   3667 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   5495 / 100000 | Speed:    1828 msg/s | 202 Accepted:   5495 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   7321 / 100000 | Speed:    1826 msg/s | 202 Accepted:   7321 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   9147 / 100000 | Speed:    1827 msg/s | 202 Accepted:   9147 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  10975 / 100000 | Speed:    1827 msg/s | 202 Accepted:  10975 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  12804 / 100000 | Speed:    1829 msg/s | 202 Accepted:  12804 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  14632 / 100000 | Speed:    1828 msg/s | 202 Accepted:  14632 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  16453 / 100000 | Speed:    1821 msg/s | 202 Accepted:  16453 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  18276 / 100000 | Speed:    1823 msg/s | 202 Accepted:  18276 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  20098 / 100000 | Speed:    1822 msg/s | 202 Accepted:  20098 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  21920 / 100000 | Speed:    1822 msg/s | 202 Accepted:  21920 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  23738 / 100000 | Speed:    1818 msg/s | 202 Accepted:  23738 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  25557 / 100000 | Speed:    1819 msg/s | 202 Accepted:  25557 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  27366 / 100000 | Speed:    1809 msg/s | 202 Accepted:  27366 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  29175 / 100000 | Speed:    1809 msg/s | 202 Accepted:  29175 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  30984 / 100000 | Speed:    1809 msg/s | 202 Accepted:  30984 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  32800 / 100000 | Speed:    1816 msg/s | 202 Accepted:  32800 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  34619 / 100000 | Speed:    1819 msg/s | 202 Accepted:  34619 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  36435 / 100000 | Speed:    1816 msg/s | 202 Accepted:  36435 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  38261 / 100000 | Speed:    1826 msg/s | 202 Accepted:  38261 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  40086 / 100000 | Speed:    1825 msg/s | 202 Accepted:  40086 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  41911 / 100000 | Speed:    1825 msg/s | 202 Accepted:  41911 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  43735 / 100000 | Speed:    1824 msg/s | 202 Accepted:  43735 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  45561 / 100000 | Speed:    1826 msg/s | 202 Accepted:  45561 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  47381 / 100000 | Speed:    1820 msg/s | 202 Accepted:  47381 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  49197 / 100000 | Speed:    1816 msg/s | 202 Accepted:  49197 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  51009 / 100000 | Speed:    1812 msg/s | 202 Accepted:  51009 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  52837 / 100000 | Speed:    1828 msg/s | 202 Accepted:  52837 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  54659 / 100000 | Speed:    1822 msg/s | 202 Accepted:  54659 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  56481 / 100000 | Speed:    1822 msg/s | 202 Accepted:  56481 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  58278 / 100000 | Speed:    1796 msg/s | 202 Accepted:  58278 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  60092 / 100000 | Speed:    1814 msg/s | 202 Accepted:  60092 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  61911 / 100000 | Speed:    1819 msg/s | 202 Accepted:  61911 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  63731 / 100000 | Speed:    1820 msg/s | 202 Accepted:  63731 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  65547 / 100000 | Speed:    1816 msg/s | 202 Accepted:  65547 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  67363 / 100000 | Speed:    1816 msg/s | 202 Accepted:  67363 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  69170 / 100000 | Speed:    1807 msg/s | 202 Accepted:  69170 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  70989 / 100000 | Speed:    1819 msg/s | 202 Accepted:  70989 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  72804 / 100000 | Speed:    1815 msg/s | 202 Accepted:  72804 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  74625 / 100000 | Speed:    1821 msg/s | 202 Accepted:  74625 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  76445 / 100000 | Speed:    1820 msg/s | 202 Accepted:  76445 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  78265 / 100000 | Speed:    1820 msg/s | 202 Accepted:  78265 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  80078 / 100000 | Speed:    1813 msg/s | 202 Accepted:  80078 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  81873 / 100000 | Speed:    1795 msg/s | 202 Accepted:  81873 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  83670 / 100000 | Speed:    1797 msg/s | 202 Accepted:  83670 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  85471 / 100000 | Speed:    1800 msg/s | 202 Accepted:  85471 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  87285 / 100000 | Speed:    1815 msg/s | 202 Accepted:  87285 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  89086 / 100000 | Speed:    1801 msg/s | 202 Accepted:  89086 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  90901 / 100000 | Speed:    1815 msg/s | 202 Accepted:  90901 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  92691 / 100000 | Speed:    1790 msg/s | 202 Accepted:  92691 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  94497 / 100000 | Speed:    1806 msg/s | 202 Accepted:  94497 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  96313 / 100000 | Speed:    1816 msg/s | 202 Accepted:  96313 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  98137 / 100000 | Speed:    1824 msg/s | 202 Accepted:  98137 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  99965 / 100000 | Speed:    1828 msg/s | 202 Accepted:  99965 | 429 Dropped: 0 | Err: 0
--------------------------------------------------------------------------------
                             BENCHMARK RESULTS
--------------------------------------------------------------------------------
 Total Requests Sent: 100000
 HTTP 202 Accepted  : 100000 (100.00%)
 HTTP 429 Dropped   : 0 (0.00%)
 Network / Err Drops: 0
 Elapsed Time       : 55.0192057s
 Realized Throughput: 1817.5 msg/sec
--------------------------------------------------------------------------------
                        LATENCY PROFILES (P50 / P90 / P99)
--------------------------------------------------------------------------------
 Minimum Latency    : 0s
 Average Latency    : 647ns
 P50 Median Latency : 0s
 P90 Latency        : 0s
 P99 Latency        : 0s
 Maximum Latency    : 828.2µs
================================================================================
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry> go run ./cmd/bench/main.go -rate=50000 -total=150000 -workers=64
================================================================================
           INDUSTRIAL IOT TELEMETRY ENGINE - STRESS BENCHMARK
================================================================================
 Target Endpoint    : http://localhost:8080/api/v1/telemetry
 Message Quota      : 150000 messages
 Target Throughput  : 50000 msg/sec
 Concurrency Workers: 64
 Host Architecture  : windows / amd64 (12 CPU cores)
--------------------------------------------------------------------------------
 Connection pool warmed up. Starting benchmark dispatch...
--------------------------------------------------------------------------------
 [LIVE] Sent:   1837 / 150000 | Speed:    1837 msg/s | 202 Accepted:   1837 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   3666 / 150000 | Speed:    1828 msg/s | 202 Accepted:   3666 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   5487 / 150000 | Speed:    1821 msg/s | 202 Accepted:   5487 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   7297 / 150000 | Speed:    1811 msg/s | 202 Accepted:   7297 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:   9105 / 150000 | Speed:    1808 msg/s | 202 Accepted:   9105 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  10923 / 150000 | Speed:    1818 msg/s | 202 Accepted:  10923 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  12746 / 150000 | Speed:    1822 msg/s | 202 Accepted:  12746 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  14549 / 150000 | Speed:    1803 msg/s | 202 Accepted:  14549 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  16356 / 150000 | Speed:    1807 msg/s | 202 Accepted:  16356 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  18164 / 150000 | Speed:    1808 msg/s | 202 Accepted:  18164 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  19981 / 150000 | Speed:    1816 msg/s | 202 Accepted:  19981 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  21805 / 150000 | Speed:    1824 msg/s | 202 Accepted:  21805 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  23630 / 150000 | Speed:    1825 msg/s | 202 Accepted:  23630 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  25454 / 150000 | Speed:    1823 msg/s | 202 Accepted:  25454 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  27267 / 150000 | Speed:    1813 msg/s | 202 Accepted:  27267 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  29089 / 150000 | Speed:    1822 msg/s | 202 Accepted:  29089 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  30918 / 150000 | Speed:    1828 msg/s | 202 Accepted:  30918 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  32744 / 150000 | Speed:    1827 msg/s | 202 Accepted:  32744 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  34565 / 150000 | Speed:    1821 msg/s | 202 Accepted:  34565 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  36389 / 150000 | Speed:    1824 msg/s | 202 Accepted:  36389 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  38217 / 150000 | Speed:    1827 msg/s | 202 Accepted:  38217 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  40043 / 150000 | Speed:    1826 msg/s | 202 Accepted:  40043 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  41869 / 150000 | Speed:    1826 msg/s | 202 Accepted:  41869 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  43693 / 150000 | Speed:    1825 msg/s | 202 Accepted:  43693 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  45522 / 150000 | Speed:    1828 msg/s | 202 Accepted:  45522 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  47347 / 150000 | Speed:    1826 msg/s | 202 Accepted:  47347 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  49172 / 150000 | Speed:    1825 msg/s | 202 Accepted:  49172 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  50996 / 150000 | Speed:    1824 msg/s | 202 Accepted:  50996 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  52823 / 150000 | Speed:    1827 msg/s | 202 Accepted:  52823 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  54646 / 150000 | Speed:    1823 msg/s | 202 Accepted:  54646 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  56475 / 150000 | Speed:    1828 msg/s | 202 Accepted:  56475 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  58300 / 150000 | Speed:    1826 msg/s | 202 Accepted:  58300 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  60125 / 150000 | Speed:    1825 msg/s | 202 Accepted:  60125 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  61949 / 150000 | Speed:    1824 msg/s | 202 Accepted:  61949 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  63771 / 150000 | Speed:    1822 msg/s | 202 Accepted:  63771 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  65596 / 150000 | Speed:    1824 msg/s | 202 Accepted:  65596 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  67419 / 150000 | Speed:    1823 msg/s | 202 Accepted:  67419 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  69245 / 150000 | Speed:    1827 msg/s | 202 Accepted:  69245 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  71065 / 150000 | Speed:    1819 msg/s | 202 Accepted:  71065 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  72890 / 150000 | Speed:    1825 msg/s | 202 Accepted:  72890 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  74714 / 150000 | Speed:    1824 msg/s | 202 Accepted:  74714 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  76538 / 150000 | Speed:    1824 msg/s | 202 Accepted:  76538 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  78361 / 150000 | Speed:    1822 msg/s | 202 Accepted:  78361 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  80189 / 150000 | Speed:    1828 msg/s | 202 Accepted:  80189 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  82011 / 150000 | Speed:    1822 msg/s | 202 Accepted:  82011 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  83825 / 150000 | Speed:    1814 msg/s | 202 Accepted:  83825 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  85646 / 150000 | Speed:    1821 msg/s | 202 Accepted:  85646 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  87472 / 150000 | Speed:    1826 msg/s | 202 Accepted:  87472 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  89297 / 150000 | Speed:    1825 msg/s | 202 Accepted:  89297 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  91121 / 150000 | Speed:    1824 msg/s | 202 Accepted:  91121 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  92948 / 150000 | Speed:    1827 msg/s | 202 Accepted:  92948 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  94772 / 150000 | Speed:    1824 msg/s | 202 Accepted:  94772 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  96595 / 150000 | Speed:    1823 msg/s | 202 Accepted:  96595 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  98420 / 150000 | Speed:    1825 msg/s | 202 Accepted:  98420 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 100239 / 150000 | Speed:    1819 msg/s | 202 Accepted: 100239 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 102066 / 150000 | Speed:    1827 msg/s | 202 Accepted: 102066 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 103891 / 150000 | Speed:    1825 msg/s | 202 Accepted: 103891 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 105716 / 150000 | Speed:    1825 msg/s | 202 Accepted: 105716 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 107539 / 150000 | Speed:    1823 msg/s | 202 Accepted: 107539 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 109364 / 150000 | Speed:    1825 msg/s | 202 Accepted: 109364 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 111189 / 150000 | Speed:    1825 msg/s | 202 Accepted: 111189 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 113015 / 150000 | Speed:    1826 msg/s | 202 Accepted: 113015 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 114838 / 150000 | Speed:    1824 msg/s | 202 Accepted: 114838 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 116663 / 150000 | Speed:    1825 msg/s | 202 Accepted: 116663 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 118487 / 150000 | Speed:    1824 msg/s | 202 Accepted: 118487 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 120307 / 150000 | Speed:    1820 msg/s | 202 Accepted: 120307 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 122130 / 150000 | Speed:    1822 msg/s | 202 Accepted: 122130 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 123955 / 150000 | Speed:    1825 msg/s | 202 Accepted: 123955 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 125773 / 150000 | Speed:    1818 msg/s | 202 Accepted: 125773 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 127579 / 150000 | Speed:    1807 msg/s | 202 Accepted: 127579 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 129389 / 150000 | Speed:    1810 msg/s | 202 Accepted: 129389 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 131208 / 150000 | Speed:    1818 msg/s | 202 Accepted: 131208 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 133021 / 150000 | Speed:    1814 msg/s | 202 Accepted: 133021 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 134848 / 150000 | Speed:    1826 msg/s | 202 Accepted: 134848 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 136671 / 150000 | Speed:    1823 msg/s | 202 Accepted: 136671 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 138497 / 150000 | Speed:    1826 msg/s | 202 Accepted: 138497 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 140318 / 150000 | Speed:    1820 msg/s | 202 Accepted: 140318 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 142139 / 150000 | Speed:    1821 msg/s | 202 Accepted: 142139 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 143963 / 150000 | Speed:    1824 msg/s | 202 Accepted: 143963 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 145790 / 150000 | Speed:    1827 msg/s | 202 Accepted: 145790 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 147613 / 150000 | Speed:    1823 msg/s | 202 Accepted: 147613 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 149441 / 150000 | Speed:    1828 msg/s | 202 Accepted: 149441 | 429 Dropped: 0 | Err: 0
--------------------------------------------------------------------------------
                             BENCHMARK RESULTS
--------------------------------------------------------------------------------
 Total Requests Sent: 150000
 HTTP 202 Accepted  : 150000 (100.00%)
 HTTP 429 Dropped   : 0 (0.00%)
 Network / Err Drops: 0
 Elapsed Time       : 1m22.3067775s
 Realized Throughput: 1822.5 msg/sec
--------------------------------------------------------------------------------
                        LATENCY PROFILES (P50 / P90 / P99)
--------------------------------------------------------------------------------
 Minimum Latency    : 0s
 Average Latency    : 850ns
 P50 Median Latency : 0s
 P90 Latency        : 0s
 P99 Latency        : 0s
 Maximum Latency    : 1.0968ms
================================================================================
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry> go run ./cmd/bench/main.go -rate=0 -total=200000 -workers=64
================================================================================
           INDUSTRIAL IOT TELEMETRY ENGINE - STRESS BENCHMARK
================================================================================
 Target Endpoint    : http://localhost:8080/api/v1/telemetry
 Message Quota      : 200000 messages
 Target Throughput  : UNLIMITED (Max Hardware Saturation)
 Concurrency Workers: 64
 Host Architecture  : windows / amd64 (12 CPU cores)
--------------------------------------------------------------------------------
 Connection pool warmed up. Starting benchmark dispatch...
--------------------------------------------------------------------------------
 [LIVE] Sent:  51536 / 200000 | Speed:   51514 msg/s | 202 Accepted:  51533 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  99671 / 200000 | Speed:   48133 msg/s | 202 Accepted:  99670 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 147574 / 200000 | Speed:   47915 msg/s | 202 Accepted: 147574 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 195519 / 200000 | Speed:   47927 msg/s | 202 Accepted: 195518 | 429 Dropped: 0 | Err: 0
--------------------------------------------------------------------------------
                             BENCHMARK RESULTS
--------------------------------------------------------------------------------
 Total Requests Sent: 200000
 HTTP 202 Accepted  : 200000 (100.00%)
 HTTP 429 Dropped   : 0 (0.00%)
 Network / Err Drops: 0
 Elapsed Time       : 4.1031432s
 Realized Throughput: 48743.1 msg/sec
--------------------------------------------------------------------------------
                        LATENCY PROFILES (P50 / P90 / P99)
--------------------------------------------------------------------------------
 Minimum Latency    : 0s
 Average Latency    : 1.163515ms
 P50 Median Latency : 1.0106ms
 P90 Latency        : 2.3018ms
 P99 Latency        : 4.7755ms
 Maximum Latency    : 17.0844ms
================================================================================
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry>
PS C:\Users\mitra\Downloads\iiot-telemetry> go run ./cmd/bench/main.go -rate=0 -total=200000 -workers=64
================================================================================
           INDUSTRIAL IOT TELEMETRY ENGINE - STRESS BENCHMARK
================================================================================
 Target Endpoint    : http://localhost:8080/api/v1/telemetry
 Message Quota      : 200000 messages
 Target Throughput  : UNLIMITED (Max Hardware Saturation)
 Concurrency Workers: 64
 Host Architecture  : windows / amd64 (12 CPU cores)
--------------------------------------------------------------------------------
 Connection pool warmed up. Starting benchmark dispatch...
--------------------------------------------------------------------------------
 [LIVE] Sent:  51607 / 200000 | Speed:   51585 msg/s | 202 Accepted:  51607 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent:  99023 / 200000 | Speed:   47420 msg/s | 202 Accepted:  99023 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 146661 / 200000 | Speed:   47651 msg/s | 202 Accepted: 146658 | 429 Dropped: 0 | Err: 0
 [LIVE] Sent: 195261 / 200000 | Speed:   48505 msg/s | 202 Accepted: 195260 | 429 Dropped: 0 | Err: 0
--------------------------------------------------------------------------------
                             BENCHMARK RESULTS
--------------------------------------------------------------------------------
 Total Requests Sent: 200000
 HTTP 202 Accepted  : 200000 (100.00%)
 HTTP 429 Dropped   : 0 (0.00%)
 Network / Err Drops: 0
 Elapsed Time       : 4.1107796s
 Realized Throughput: 48652.6 msg/sec
--------------------------------------------------------------------------------
                        LATENCY PROFILES (P50 / P90 / P99)
--------------------------------------------------------------------------------
 Minimum Latency    : 0s
 Average Latency    : 1.159648ms
 P50 Median Latency : 1.0108ms
 P90 Latency        : 2.305ms
 P99 Latency        : 4.7921ms
 Maximum Latency    : 18.0876ms
================================================================================
```
