// Package main implements a user service that manages user accounts,
// authentication, and authorization with proper security measures.
package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/ramisback/istio-rate-limiter/user-service/internal/repository"
	"github.com/ramisback/istio-rate-limiter/user-service/internal/service"
	pb "github.com/ramisback/istio-rate-limiter/user-service/proto"
)

var (
	// Prometheus metrics for monitoring user service operations
	userOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "user_operations_total",
			Help: "Total number of user operations",
		},
		[]string{"operation", "status"},
	)

	authAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_attempts_total",
			Help: "Total number of authentication attempts",
		},
		[]string{"status"},
	)

	operationLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "operation_latency_seconds",
			Help:    "Operation latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	activeSessions = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_sessions",
			Help: "Number of active user sessions",
		},
		[]string{"user_type"},
	)
)

// User represents a user account
type User struct {
	ID           string    // Unique user identifier
	Username     string    // Username for login
	Email        string    // User's email address
	PasswordHash string    // Hashed password
	Role         string    // User role (admin, user, etc.)
	CreatedAt    time.Time // Account creation timestamp
	LastLogin    time.Time // Last login timestamp
}

// UserService manages user accounts and authentication
type UserService struct {
	redis  *redis.Client // Redis client for session storage
	logger *zap.Logger   // Structured logger
	jwtKey []byte        // JWT signing key
}

// Session represents a user session
type Session struct {
	UserID    string    // Associated user ID
	Token     string    // Session token
	ExpiresAt time.Time // Session expiration time
}

// NewUserService creates a new user service instance
func NewUserService(logger *zap.Logger) (*UserService, error) {
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
	jwtKey := make([]byte, 32)
	if _, err := rand.Read(jwtKey); err != nil {
		return nil, fmt.Errorf("failed to generate JWT key: %v", err)
	}

	return &UserService{
		redis:  redisClient,
		logger: logger,
		jwtKey: jwtKey,
	}, nil
}

// CreateUser creates a new user account
func (s *UserService) CreateUser(ctx context.Context, user *User) error {
	start := time.Now()
	defer func() {
		operationLatency.WithLabelValues("create_user").Observe(time.Since(start).Seconds())
	}()

	// Validate user data
	if err := s.validateUser(user); err != nil {
		userOperations.WithLabelValues("create", "validation_error").Inc()
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Check if username exists
	exists, err := s.redis.Exists(ctx, fmt.Sprintf("user:%s", user.Username)).Result()
	if err != nil {
		userOperations.WithLabelValues("create", "redis_error").Inc()
		return status.Error(codes.Internal, "failed to check username")
	}
	if exists == 1 {
		userOperations.WithLabelValues("create", "duplicate_username").Inc()
		return status.Error(codes.AlreadyExists, "username already exists")
	}

	// Store user data
	key := fmt.Sprintf("user:%s", user.Username)
	if err := s.redis.HSet(ctx, key, map[string]interface{}{
		"id":            user.ID,
		"email":         user.Email,
		"password_hash": user.PasswordHash,
		"role":          user.Role,
		"created_at":    user.CreatedAt.Unix(),
		"last_login":    user.LastLogin.Unix(),
	}).Err(); err != nil {
		userOperations.WithLabelValues("create", "redis_error").Inc()
		return status.Error(codes.Internal, "failed to create user")
	}

	userOperations.WithLabelValues("create", "success").Inc()
	return nil
}

// AuthenticateUser authenticates a user and creates a session
func (s *UserService) AuthenticateUser(ctx context.Context, username, password string) (*Session, error) {
	start := time.Now()
	defer func() {
		operationLatency.WithLabelValues("authenticate").Observe(time.Since(start).Seconds())
	}()

	// Get user data
	key := fmt.Sprintf("user:%s", username)
	userData, err := s.redis.HGetAll(ctx, key).Result()
	if err != nil {
		authAttempts.WithLabelValues("redis_error").Inc()
		return nil, status.Error(codes.Internal, "failed to get user data")
	}
	if len(userData) == 0 {
		authAttempts.WithLabelValues("user_not_found").Inc()
		return nil, status.Error(codes.NotFound, "user not found")
	}

	// Verify password
	if !s.verifyPassword(password, userData["password_hash"]) {
		authAttempts.WithLabelValues("invalid_password").Inc()
		return nil, status.Error(codes.Unauthenticated, "invalid password")
	}

	// Create session
	session := &Session{
		UserID:    userData["id"],
		Token:     s.generateToken(userData["id"], userData["role"]),
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Store session
	sessionKey := fmt.Sprintf("session:%s", session.Token)
	if err := s.redis.Set(ctx, sessionKey, session.UserID, 24*time.Hour).Err(); err != nil {
		authAttempts.WithLabelValues("session_error").Inc()
		return nil, status.Error(codes.Internal, "failed to create session")
	}

	// Update last login
	if err := s.redis.HSet(ctx, key, "last_login", time.Now().Unix()).Err(); err != nil {
		s.logger.Error("failed to update last login",
			zap.Error(err),
			zap.String("username", username),
		)
	}

	authAttempts.WithLabelValues("success").Inc()
	activeSessions.WithLabelValues(userData["role"]).Inc()
	return session, nil
}

// validateUser validates user data
func (s *UserService) validateUser(user *User) error {
	if user.Username == "" {
		return fmt.Errorf("username is required")
	}
	if user.Email == "" {
		return fmt.Errorf("email is required")
	}
	if user.PasswordHash == "" {
		return fmt.Errorf("password is required")
	}
	if user.Role == "" {
		return fmt.Errorf("role is required")
	}
	return nil
}

// verifyPassword verifies a password against its hash
func (s *UserService) verifyPassword(password, hash string) bool {
	// Implement secure password verification
	// This is a placeholder - use a proper password hashing library
	return password == hash
}

// generateToken generates a JWT token for a user
func (s *UserService) generateToken(userID, role string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(s.jwtKey)
	if err != nil {
		s.logger.Error("failed to generate token",
			zap.Error(err),
			zap.String("user_id", userID),
		)
		return ""
	}

	return tokenString
}

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
		logger.Fatal("failed to connect to Redis",
			zap.Error(err),
		)
	}

	// Create user repository
	userRepo := repository.NewUserRepository(redisClient)

	// Create user service
	userSvc := service.NewUserService(logger, userRepo)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register user service
	pb.RegisterUserServiceServer(grpcServer, service.NewGRPCServer(userSvc, logger))

	// Enable reflection for debugging
	reflection.Register(grpcServer)

	// Initialize gRPC server
	lis, err := net.Listen("tcp", ":8083")
	if err != nil {
		logger.Fatal("failed to listen",
			zap.Error(err),
			zap.String("address", ":8083"),
		)
	}

	// Start Prometheus metrics endpoint
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":9092", nil); err != nil {
			logger.Error("metrics server error",
				zap.Error(err),
			)
		}
	}()

	// Log service startup
	logger.Info("user service starting",
		zap.String("address", ":8083"),
	)

	// Start gRPC server
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("failed to serve",
			zap.Error(err),
		)
	}
}
