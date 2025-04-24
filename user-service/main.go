// Package main implements a user service that manages user accounts,
// authentication, and authorization with proper security measures.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

var (
	// Prometheus metrics
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "user_service_request_duration_seconds",
			Help:    "Duration of user service requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint", "method", "status"},
	)

	requestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_service_requests_total",
			Help: "Total number of requests to user service",
		},
		[]string{"endpoint", "method"},
	)
)

// User represents a user account
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Password string `json:"-"` // Password is never sent in JSON responses
	Role     string `json:"role"`
}

// UserService manages user accounts and authentication
type UserService struct {
	redis  *redis.Client
	jwtKey []byte
}

// NewUserService creates a new user service instance
func NewUserService() (*UserService, error) {
	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // Set if required
		DB:       0,  // Use default DB
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	// Generate JWT key
	jwtKey := []byte("your-secret-key") // In production, use a secure key

	return &UserService{
		redis:  redisClient,
		jwtKey: jwtKey,
	}, nil
}

// loggingMiddleware logs all incoming requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom response writer to capture the status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Log the incoming request
		log.Printf("Incoming request: %s %s", r.Method, r.URL.Path)

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Log the response
		duration := time.Since(start).Seconds()
		log.Printf("Request completed: %s %s - Status: %d - Duration: %.3fs",
			r.Method, r.URL.Path, rw.statusCode, duration)

		// Record metrics
		requestDuration.WithLabelValues(r.URL.Path, r.Method, fmt.Sprintf("%d", rw.statusCode)).Observe(duration)
		requestCounter.WithLabelValues(r.URL.Path, r.Method).Inc()
	})
}

// responseWriter is a custom response writer that captures the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (s *UserService) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var user User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Store user in Redis
	userKey := fmt.Sprintf("user:%s", user.Email)
	if err := s.redis.HSet(r.Context(), userKey, map[string]interface{}{
		"id":       user.ID,
		"email":    user.Email,
		"password": user.Password,
		"role":     user.Role,
	}).Err(); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (s *UserService) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var creds struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user from Redis
	userKey := fmt.Sprintf("user:%s", creds.Email)
	userData, err := s.redis.HGetAll(r.Context(), userKey).Result()
	if err != nil || len(userData) == 0 {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Check password
	if userData["password"] != creds.Password {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userData["id"],
		"role":    userData["role"],
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(s.jwtKey)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenString,
	})
}

func (s *UserService) ValidateToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtKey, nil
	})
}

// DummyService provides endpoints with different response times
type DummyService struct{}

func NewDummyService() *DummyService {
	return &DummyService{}
}

// FastEndpoint responds quickly (10ms)
func (s *DummyService) FastEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	time.Sleep(10 * time.Millisecond)
	json.NewEncoder(w).Encode(map[string]string{"message": "Fast response"})
}

// MediumEndpoint responds with medium latency (100ms)
func (s *DummyService) MediumEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	time.Sleep(100 * time.Millisecond)
	json.NewEncoder(w).Encode(map[string]string{"message": "Medium response"})
}

// SlowEndpoint responds slowly (500ms)
func (s *DummyService) SlowEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	time.Sleep(500 * time.Millisecond)
	json.NewEncoder(w).Encode(map[string]string{"message": "Slow response"})
}

// VerySlowEndpoint responds very slowly (1s)
func (s *DummyService) VerySlowEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	time.Sleep(1 * time.Second)
	json.NewEncoder(w).Encode(map[string]string{"message": "Very slow response"})
}

func main() {
	userService, err := NewUserService()
	if err != nil {
		log.Fatalf("Failed to create user service: %v", err)
	}

	service := NewDummyService()

	// Create a new mux router
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/users", userService.CreateUser)
	mux.HandleFunc("/login", userService.Login)

	// Register routes with different response times
	mux.HandleFunc("/fast", service.FastEndpoint)          // 10ms
	mux.HandleFunc("/medium", service.MediumEndpoint)      // 100ms
	mux.HandleFunc("/slow", service.SlowEndpoint)          // 500ms
	mux.HandleFunc("/very-slow", service.VerySlowEndpoint) // 1s

	// Add Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Wrap the mux with our logging middleware
	handler := loggingMiddleware(mux)

	log.Printf("User service starting on :8083")
	log.Printf("Available endpoints:")
	log.Printf("  GET /fast      - 10ms response time")
	log.Printf("  GET /medium    - 100ms response time")
	log.Printf("  GET /slow      - 500ms response time")
	log.Printf("  GET /very-slow - 1s response time")
	log.Printf("  GET /metrics   - Prometheus metrics")

	if err := http.ListenAndServe(":8083", handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
