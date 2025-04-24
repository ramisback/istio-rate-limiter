// Package main implements a rate limiting service that integrates with Istio's rate limiting
// functionality. It provides distributed rate limiting using Redis for state management
// and includes local caching for performance optimization.
package main

import (
	"context"  // For context management
	"fmt"      // For formatted I/O
	"log"      // For logging
	"net"      // For network operations
	"net/http" // For HTTP server

	// For string operations
	// For environment variables
	// For string conversions
	"time" // For time operations

	"github.com/dgraph-io/ristretto" // For local caching
	// Envoy rate limit service
	ratelimit "github.com/envoyproxy/go-control-plane/envoy/extensions/common/ratelimit/v3"
	envoy "github.com/envoyproxy/go-control-plane/envoy/service/ratelimit/v3"
	"github.com/prometheus/client_golang/prometheus"          // Prometheus metrics
	"github.com/prometheus/client_golang/prometheus/promauto" // Prometheus auto-registration
	"github.com/prometheus/client_golang/prometheus/promhttp" // Prometheus HTTP handler
	"github.com/redis/go-redis/v9"                            // Redis client
	"go.uber.org/zap"                                         // Structured logging
	"google.golang.org/grpc"                                  // gRPC server
	"google.golang.org/grpc/metadata"                         // gRPC metadata
	"google.golang.org/grpc/reflection"                       // gRPC reflection
)

// Context keys for tracing
type contextKey string

const (
	requestIDKey contextKey = "x-request-id"
	traceIDKey   contextKey = "x-b3-traceid"
	spanIDKey    contextKey = "x-b3-spanid"
)

// Prometheus metrics for monitoring rate limiting operations
var (
	// rateLimitRequests tracks the total number of rate limit requests processed,
	// labeled by status (success/error), type (request), and error message
	rateLimitRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rate_limit_requests_total",
			Help: "Total number of rate limit requests processed",
		},
		[]string{"status", "type", "error"},
	)

	// rateLimitLatency measures the latency of rate limit checks in seconds,
	// using standard Prometheus buckets for histogram analysis
	rateLimitLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rate_limit_latency_seconds",
			Help:    "Rate limit request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type"},
	)

	// redisErrors tracks Redis operation failures,
	// labeled by the type of operation that failed
	redisErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_errors_total",
			Help: "Total number of Redis errors",
		},
		[]string{"operation"},
	)
)

// CompanyLimits defines rate limits for a specific company
type CompanyLimits struct {
	RequestsPerMinute int // Maximum number of requests allowed per minute
}

// RateLimitConfig defines rate limits for different types of requests
type RateLimitConfig struct {
	IPLimit      int64
	PathLimit    int64
	CompanyLimit int64
	UserLimit    int64
	Window       time.Duration
}

// RateLimitServer implements the Envoy rate limit service interface
// and manages the rate limiting state and operations
type RateLimitServer struct {
	envoy.UnimplementedRateLimitServiceServer
	localCache   *ristretto.Cache             // Local cache for rate limit decisions
	redis        *redis.ClusterClient         // Redis cluster client for distributed state
	updateQueue  chan *envoy.RateLimitRequest // Channel for async updates
	workerPool   *UpdateWorkerPool            // Pool of workers for processing updates
	ipLimit      int64                        // Rate limit for IP-based limiting
	pathLimit    int64                        // Rate limit for path-based limiting
	companyLimit int64                        // Rate limit for company-based limiting
	userLimit    int64                        // Rate limit for user-based limiting
	window       time.Duration                // Time window for rate limiting
	metrics      *prometheus.CounterVec       // Prometheus metrics
	logger       *zap.Logger                  // Structured logger
}

// RateLimitRequest represents a rate limit check request
type RateLimitRequest struct {
	IP        string // Client IP address
	CompanyID string // Company identifier from JWT
	Path      string // Request path
	Method    string // HTTP method
}

// UpdateWorkerPool manages a pool of workers for processing rate limit updates
// and ensures efficient batch processing of Redis operations
type UpdateWorkerPool struct {
	workers []*UpdateWorker              // List of worker goroutines
	queue   chan *envoy.RateLimitRequest // Shared queue for updates
	logger  *zap.Logger                  // Structured logger
}

// UpdateWorker processes rate limit updates in batches
// and handles the actual Redis operations
type UpdateWorker struct {
	queue  chan *envoy.RateLimitRequest // Queue for receiving updates
	redis  *redis.ClusterClient         // Redis client for state updates
	buffer []*envoy.RateLimitRequest    // Buffer for batching updates
	logger *zap.Logger                  // Structured logger
}

// NewRateLimitServer creates and initializes a new rate limit server
// with all necessary components and configurations
func NewRateLimitServer() (*RateLimitServer, error) {
	// Initialize structured logger for production use
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %v", err)
	}

	// Initialize local cache with optimized settings for high throughput
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // Track frequency of 10M keys
		MaxCost:     1 << 30, // Maximum cache size (1GB)
		BufferItems: 64,      // Keys per Get buffer
		OnEvict: func(item *ristretto.Item) {
			logger.Debug("cache item evicted",
				zap.String("key", fmt.Sprintf("%v", item.Key)),
				zap.Int64("cost", item.Cost),
			)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %v", err)
	}

	// Define Redis cluster addresses for high availability
	redisAddrs := []string{
		"redis-cluster-0.redis:6379",
		"redis-cluster-1.redis:6379",
		"redis-cluster-2.redis:6379",
	}

	// Initialize Redis cluster client with connection settings
	rdb := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        redisAddrs,
		ReadTimeout:  time.Second, // Timeout for read operations
		WriteTimeout: time.Second, // Timeout for write operations
		MaxRedirects: 3,           // Maximum number of redirects
		OnConnect: func(ctx context.Context, cn *redis.Conn) error {
			logger.Info("connected to Redis node",
				zap.String("addr", fmt.Sprintf("%v", cn)),
			)
			return nil
		},
	})

	// Test Redis connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	// Initialize worker pool for processing updates
	pool := NewUpdateWorkerPool(10, rdb, logger)

	// Create and configure rate limit server with updated limits
	server := &RateLimitServer{
		localCache:   cache,
		redis:        rdb,
		updateQueue:  make(chan *envoy.RateLimitRequest, 10000),
		workerPool:   pool,
		ipLimit:      1000,        // 1000 requests per window per IP
		pathLimit:    500,         // 500 requests per window per path
		companyLimit: 10000,       // 10000 requests per window per company
		userLimit:    100,         // 100 requests per window per user
		window:       time.Minute, // 1-minute window
		metrics:      rateLimitRequests,
		logger:       logger,
	}

	return server, nil
}

// NewUpdateWorkerPool creates a new pool of update workers
// with the specified size and Redis client
func NewUpdateWorkerPool(size int, redis *redis.ClusterClient, logger *zap.Logger) *UpdateWorkerPool {
	pool := &UpdateWorkerPool{
		workers: make([]*UpdateWorker, size),
		queue:   make(chan *envoy.RateLimitRequest, 10000), // Buffer for 10k requests
		logger:  logger,
	}

	// Initialize and start workers
	for i := 0; i < size; i++ {
		pool.workers[i] = &UpdateWorker{
			queue:  pool.queue,
			redis:  redis,
			buffer: make([]*envoy.RateLimitRequest, 0, 100), // Buffer for batching
			logger: logger,
		}
		go pool.workers[i].Start()
	}

	return pool
}

// Start begins processing updates in the worker
// and manages the update buffer and Redis operations
func (w *UpdateWorker) Start() {
	ticker := time.NewTicker(100 * time.Millisecond) // Flush every 100ms
	defer ticker.Stop()

	for {
		select {
		case req := <-w.queue:
			w.buffer = append(w.buffer, req)
			if len(w.buffer) >= 100 { // Flush when buffer is full
				w.flush()
			}
		case <-ticker.C:
			if len(w.buffer) > 0 { // Flush on ticker if buffer not empty
				w.flush()
			}
		}
	}
}

// flush writes buffered updates to Redis
// and handles any errors that occur during the operation
func (w *UpdateWorker) flush() {
	if len(w.buffer) == 0 {
		return
	}

	pipe := w.redis.Pipeline()
	// Batch increment counters for each request
	for _, req := range w.buffer {
		// Extract IP and company ID from descriptors
		ip := ""
		companyID := ""
		for _, entry := range req.Descriptors {
			for _, kv := range entry.Entries {
				if kv.Key == "ip" {
					ip = kv.Value
				} else if kv.Key == "company_id" {
					companyID = kv.Value
				}
			}
		}

		// Increment counters for each type of limit
		if ip != "" {
			pipe.Incr(context.Background(), fmt.Sprintf("ip:%s", ip))
		}
		if companyID != "" {
			pipe.Incr(context.Background(), fmt.Sprintf("company:%s", companyID))
		}
		if ip != "" && companyID != "" {
			pipe.Incr(context.Background(), fmt.Sprintf("combined:%s:%s", ip, companyID))
		}
	}

	// Execute pipeline and handle errors
	if _, err := pipe.Exec(context.Background()); err != nil {
		redisErrors.WithLabelValues("pipeline_exec").Inc()
		w.logger.Error("failed to execute Redis pipeline",
			zap.Error(err),
			zap.Int("batch_size", len(w.buffer)),
		)
	}

	w.buffer = w.buffer[:0] // Clear buffer
}

// ShouldRateLimit implements the Envoy rate limit service interface
// and processes rate limit requests
func (s *RateLimitServer) ShouldRateLimit(ctx context.Context, req *envoy.RateLimitRequest) (*envoy.RateLimitResponse, error) {
	start := time.Now()
	defer func() {
		rateLimitLatency.WithLabelValues("request").Observe(time.Since(start).Seconds())
	}()

	// Extract request metadata for tracing
	requestID := ctx.Value(requestIDKey)
	traceID := ctx.Value(traceIDKey)
	spanID := ctx.Value(spanIDKey)

	// Log request details
	s.logger.Info("processing rate limit request",
		zap.Any("request_id", requestID),
		zap.Any("trace_id", traceID),
		zap.Any("span_id", spanID),
	)

	// Initialize response
	response := &envoy.RateLimitResponse{
		OverallCode: envoy.RateLimitResponse_OK,
		Statuses:    make([]*envoy.RateLimitResponse_DescriptorStatus, len(req.Descriptors)),
	}

	// Process each descriptor
	for i, descriptor := range req.Descriptors {
		status := &envoy.RateLimitResponse_DescriptorStatus{
			Code:           envoy.RateLimitResponse_OK,
			CurrentLimit:   nil,
			LimitRemaining: 0,
		}

		// Check rate limits
		limit, remaining, err := s.checkRateLimit(descriptor)
		if err != nil {
			s.logger.Error("error checking rate limit",
				zap.Error(err),
				zap.Any("descriptor", descriptor),
			)
			rateLimitRequests.WithLabelValues("error", "request", err.Error()).Inc()
			status.Code = envoy.RateLimitResponse_OVER_LIMIT
			continue
		}

		// Set limit information if applicable
		if limit > 0 {
			status.CurrentLimit = &envoy.RateLimitResponse_RateLimit{
				RequestsPerUnit: uint32(limit),
				Unit:            envoy.RateLimitResponse_RateLimit_MINUTE,
			}
			status.LimitRemaining = uint32(remaining)
		}

		response.Statuses[i] = status
	}

	// Record success metric
	rateLimitRequests.WithLabelValues("success", "request", "").Inc()
	return response, nil
}

// checkRateLimit checks if a request should be rate limited based on its descriptors
func (s *RateLimitServer) checkRateLimit(descriptor *ratelimit.RateLimitDescriptor) (int, int, error) {
	var limit int64
	var key string

	// Extract rate limit key and limit based on descriptor
	for _, entry := range descriptor.Entries {
		switch entry.Key {
		case "remote_address":
			limit = s.ipLimit
			key = fmt.Sprintf("ip:%s", entry.Value)
		case "path":
			limit = s.pathLimit
			key = fmt.Sprintf("path:%s", entry.Value)
		case "company_id":
			limit = s.companyLimit
			key = fmt.Sprintf("company:%s", entry.Value)
		case "user_id":
			limit = s.userLimit
			key = fmt.Sprintf("user:%s", entry.Value)
		}
	}

	if key == "" {
		return 0, 0, fmt.Errorf("no valid rate limit key found in descriptor")
	}

	// Check local cache first
	if val, found := s.localCache.Get(key); found {
		count := val.(int64)
		if count >= limit {
			return int(count), int(limit), nil
		}
	}

	// Check Redis for distributed rate limiting
	ctx := context.Background()
	count, err := s.redis.Incr(ctx, key).Result()
	if err != nil {
		redisErrors.WithLabelValues("incr").Inc()
		return 0, 0, fmt.Errorf("redis error: %v", err)
	}

	// Set expiration if this is the first request
	if count == 1 {
		s.redis.Expire(ctx, key, s.window)
	}

	// Update local cache
	s.localCache.Set(key, count, 1)

	// Return current count and limit
	return int(count), int(limit), nil
}

// main initializes and runs the rate limit service
func main() {
	// Initialize structured logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Printf("failed to sync logger: %v", err)
		}
	}()

	// Initialize gRPC server
	lis, err := net.Listen("tcp", ":8081")
	if err != nil {
		logger.Fatal("failed to listen",
			zap.Error(err),
			zap.String("address", ":8081"),
		)
	}

	// Create gRPC server with tracing interceptor
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcTracingInterceptor),
	)

	// Register rate limit service
	server, err := NewRateLimitServer()
	if err != nil {
		logger.Fatal("failed to create rate limit server",
			zap.Error(err),
		)
	}
	envoy.RegisterRateLimitServiceServer(grpcServer, server)

	// Enable reflection for debugging
	reflection.Register(grpcServer)

	// Start Prometheus metrics endpoint
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":9090", nil); err != nil {
			logger.Error("metrics server error",
				zap.Error(err),
			)
		}
	}()

	// Log service startup
	logger.Info("rate limit service starting",
		zap.String("address", ":8081"),
	)

	// Start gRPC server
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("failed to serve",
			zap.Error(err),
		)
	}
}

// grpcTracingInterceptor adds tracing headers to gRPC context
// for distributed tracing support
func grpcTracingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		// Add request ID to context
		if requestIDs := md.Get(string(requestIDKey)); len(requestIDs) > 0 {
			ctx = context.WithValue(ctx, requestIDKey, requestIDs[0])
		}

		// Add trace ID to context
		if traceIDs := md.Get(string(traceIDKey)); len(traceIDs) > 0 {
			ctx = context.WithValue(ctx, traceIDKey, traceIDs[0])
		}

		// Add span ID to context
		if spanIDs := md.Get(string(spanIDKey)); len(spanIDs) > 0 {
			ctx = context.WithValue(ctx, spanIDKey, spanIDs[0])
		}
	}

	return handler(ctx, req)
}
