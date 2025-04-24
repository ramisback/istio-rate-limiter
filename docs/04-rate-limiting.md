# Rate Limiting Implementation

## Overview

This document describes the rate limiting implementation in the Istio Rate Limiter project. The system provides multi-level rate limiting capabilities using Istio, Envoy, and a custom rate limit service.

## Rate Limiting Architecture

### High-Level Flow
```
┌───────────────┐     ┌──────────────────┐     ┌─────────────────┐
│ Client        │────►│ Istio Gateway    │────►│ Rate Limit      │
│ Request       │     │                  │     │ Filter          │
└───────────────┘     └──────────────────┘     └─────────────────┘
                                                       │
                                                       ▼
┌───────────────┐     ┌──────────────────┐     ┌─────────────────┐
│ Redis         │◄────│ Rate Limit       │◄────│ Rate Limit      │
│ Storage       │     │ Service          │     │ Decision        │
└───────────────┘     └──────────────────┘     └─────────────────┘
                                                       │
                                                       ▼
┌───────────────┐     ┌──────────────────┐     ┌─────────────────┐
│ User          │◄────│ JWT              │◄────│ Forward         │
│ Service       │     │ Filter           │     │ Request         │
└───────────────┘     └──────────────────┘     └─────────────────┘
```

### Request Processing Flow
1. Client sends request to Istio Gateway
2. Rate Limit Filter intercepts request
3. Rate Limit Service checks limits against Redis
4. If allowed, request proceeds to JWT Filter
5. JWT Filter validates token
6. Request reaches User Service
7. Counters are updated in Redis

## Rate Limiting Types

### 1. IP-Based Rate Limiting
- Limits requests based on client IP
- Uses X-Forwarded-For header
- Default: 1000 requests per minute
- Configurable via environment variables

### 2. Company-Based Rate Limiting
- Limits requests based on company ID
- Extracted from JWT token
- Default: 10000 requests per minute per company
- Configurable per company

### 3. Global Rate Limiting
- Overall system-wide limits
- Prevents system overload
- Default: 100000 requests per minute
- Configurable via configuration

## Implementation Details

### 1. Rate Limit Service

#### Rate Limit Logic
```go
type RateLimiter struct {
    window       time.Duration
    ipLimit      int
    companyLimit int
    globalLimit  int
    redis        *redis.Client
}

func (rl *RateLimiter) CheckLimit(descriptor RateLimit) (bool, error) {
    // Check global limit first
    if !rl.checkGlobalLimit() {
        return false, nil
    }

    // Check specific limits based on descriptor
    switch descriptor.Type {
    case IP:
        return rl.checkIPLimit(descriptor.Value)
    case Company:
        return rl.checkCompanyLimit(descriptor.Value)
    default:
        return true, nil
    }
}
```

## Rate Limit Storage

### 1. Redis Implementation
- Uses Redis Sorted Sets for sliding window
- Atomic operations for counter updates
- Automatic key expiration
- Cluster support for scalability

### 2. Storage Schema
```
Key Format:
- IP rate limit: "ip:{ip}:{window}"
- Company rate limit: "company:{id}:{window}"
- Global rate limit: "global:{window}"

Value Format:
- Sorted set of timestamps
- Score: Unix timestamp
- Member: Request ID
```

### 3. Cleanup Strategy
- Automatic key expiration
- Background cleanup job
- Configurable retention period
- Memory usage monitoring

## Performance Considerations

### 1. Redis Optimization
- Connection pooling
- Pipeline commands
- Batch operations
- Key expiration strategy

### 2. Rate Limit Service
- In-memory caching
- Goroutine pooling
- Efficient counter implementation
- Error handling and fallbacks

### 3. Envoy Configuration
- Timeout settings
- Circuit breaking
- Retry policy
- Buffer settings

## Monitoring and Metrics

### 1. Rate Limit Metrics
```
# Prometheus metrics
rate_limit_requests_total{type="ip"} 
rate_limit_requests_total{type="company"}
rate_limit_requests_total{type="global"}
rate_limit_exceeded_total{type="ip"}
rate_limit_exceeded_total{type="company"}
rate_limit_exceeded_total{type="global"}
```

### 2. Redis Metrics
- Connection pool stats
- Command latency
- Memory usage
- Key statistics

### 3. Service Metrics
- Request latency
- Error rates
- Cache hit rates

## For detailed configuration options, see the [Configuration Guide](05-configuration.md). 