package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	// Use environment variable for service URL with fallback
	defaultBaseURL = "http://user-service.default.svc.cluster.local"
)

var (
	requestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "loadtest_requests_total",
			Help: "Total number of requests made",
		},
		[]string{"status", "company"},
	)
	requestLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "loadtest_request_duration_seconds",
			Help:    "Request latency distribution",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"company"},
	)
)

type Config struct {
	targetURL     string
	rps           int
	duration      time.Duration
	numCompanies  int
	jwtSecret     string
	concurrency   int
	enableMetrics bool
	metricsPort   int
}

type Result struct {
	Endpoint     string
	StatusCode   int
	ResponseTime time.Duration
	Error        error
}

var baseURL string

func main() {
	config := parseFlags()

	if config.enableMetrics {
		go serveMetrics(config.metricsPort)
	}

	// Get service URL from environment or use default
	baseURL = os.Getenv("SERVICE_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	fmt.Printf("Starting load test for %v with %d concurrent workers\n", config.duration, config.concurrency)
	fmt.Printf("Target service URL: %s\n", baseURL)

	// Test connection before starting load test
	if err := testConnection(); err != nil {
		fmt.Printf("Error: Cannot connect to service: %v\n", err)
		fmt.Println("Please check if:")
		fmt.Println("1. The service is running (kubectl get pods -l app=user-service)")
		fmt.Println("2. The service is accessible (kubectl get svc user-service)")
		fmt.Println("3. You're running this from within the cluster or have proper port-forwarding")
		os.Exit(1)
	}

	runLoadTest(config)
}

func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.targetURL, "url", "http://localhost:8080", "Target URL")
	flag.IntVar(&config.rps, "rps", 100, "Requests per second")
	flag.DurationVar(&config.duration, "duration", 5*time.Minute, "Test duration")
	flag.IntVar(&config.numCompanies, "companies", 3, "Number of companies")
	flag.StringVar(&config.jwtSecret, "jwt-secret", "secret", "JWT signing secret")
	flag.IntVar(&config.concurrency, "concurrency", 10, "Number of concurrent workers")
	flag.BoolVar(&config.enableMetrics, "metrics", true, "Enable Prometheus metrics")
	flag.IntVar(&config.metricsPort, "metrics-port", 9090, "Metrics port")

	flag.Parse()
	return config
}

func serveMetrics(port int) {
	http.Handle("/metrics", promhttp.Handler())
	log.Printf("Starting metrics server on :%d", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatalf("Failed to start metrics server: %v", err)
	}
}

func testConnection() error {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(baseURL + "/health")
	if err != nil {
		return fmt.Errorf("failed to connect to service: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("service returned non-200 status code: %d", resp.StatusCode)
	}

	return nil
}

func runLoadTest(config *Config) {
	ticker := time.NewTicker(time.Second / time.Duration(config.rps))
	defer ticker.Stop()

	done := make(chan bool)
	go func() {
		time.Sleep(config.duration)
		done <- true
	}()

	// Create worker pool
	jobs := make(chan int, config.rps)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < config.concurrency; i++ {
		wg.Add(1)
		go worker(config, jobs, &wg)
	}

	log.Printf("Starting load test: %d RPS for %v", config.rps, config.duration)
	requestCount := 0

	for {
		select {
		case <-done:
			close(jobs)
			wg.Wait()
			log.Printf("Load test completed. Total requests: %d", requestCount)
			return
		case <-ticker.C:
			requestCount++
			jobs <- requestCount
		}
	}
}

func worker(config *Config, jobs <-chan int, wg *sync.WaitGroup) {
	defer wg.Done()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for range jobs {
		companyID := fmt.Sprintf("company%d", time.Now().UnixNano()%int64(config.numCompanies))
		token := generateJWT(companyID, config.jwtSecret)

		start := time.Now()
		status := makeRequest(client, config.targetURL, companyID, token)
		duration := time.Since(start)

		if config.enableMetrics {
			requestsTotal.WithLabelValues(status, companyID).Inc()
			requestLatency.WithLabelValues(companyID).Observe(duration.Seconds())
		}
	}
}

func makeRequest(client *http.Client, url, companyID, token string) string {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return "error"
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Company-ID", companyID)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making request: %v", err)
		return "error"
	}
	defer resp.Body.Close()

	return fmt.Sprintf("%d", resp.StatusCode)
}

func generateJWT(companyID, secret string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":        companyID,
		"company_id": companyID,
		"exp":        time.Now().Add(time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Printf("Error generating JWT: %v", err)
		return ""
	}

	return tokenString
}
