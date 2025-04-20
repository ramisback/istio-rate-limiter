# API Reference

## Overview

This document provides detailed information about the API endpoints:
- User Service API
- Rate Limit Service API
- Authentication endpoints
- Metrics endpoints

## User Service API

### Health Check

```http
GET /health
```

**Response**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "timestamp": "2024-02-20T10:15:30Z"
}
```

### Users Endpoint

```http
GET /users
```

**Headers**
```http
Authorization: Bearer <jwt-token>
X-Company-ID: company1
```

**Response**
```json
{
  "users": [
    {
      "id": "user1",
      "name": "John Doe",
      "email": "john@example.com",
      "company": "company1",
      "role": "user"
    }
  ],
  "metadata": {
    "total": 100,
    "page": 1,
    "per_page": 10
  }
}
```

### Authentication

```http
POST /auth/token
```

**Request**
```json
{
  "company_id": "company1",
  "username": "user1",
  "password": "password123"
}
```

**Response**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 3600,
  "token_type": "Bearer"
}
```

## Rate Limit Service API

### Check Rate Limit

```http
POST /v1/ratelimit/check
```

**Request**
```json
{
  "domain": "production",
  "descriptors": [
    {
      "key": "remote_address",
      "value": "192.168.1.1"
    },
    {
      "key": "company_id",
      "value": "company1"
    }
  ]
}
```

**Response**
```json
{
  "overall_code": "OK",
  "statuses": [
    {
      "code": "OK",
      "current_limit": 100,
      "limit_remaining": 95,
      "duration_until_reset": 55
    }
  ]
}
```

### Rate Limit Configuration

```http
GET /config/rate-limits
```

**Response**
```json
{
  "domains": [
    {
      "name": "production",
      "descriptors": [
        {
          "key": "remote_address",
          "rate_limit": {
            "unit": "minute",
            "requests_per_unit": 30
          }
        },
        {
          "key": "company_id",
          "value": "company1",
          "rate_limit": {
            "unit": "minute",
            "requests_per_unit": 100
          }
        }
      ]
    }
  ]
}
```

## Metrics Endpoints

### Prometheus Metrics

```http
GET /metrics
```

**Response**
```text
# HELP rate_limit_service_requests_total Total number of rate limit requests
# TYPE rate_limit_service_requests_total counter
rate_limit_service_requests_total{status="allowed"} 1234
rate_limit_service_requests_total{status="denied"} 567

# HELP rate_limit_service_request_duration_seconds Request latency in seconds
# TYPE rate_limit_service_request_duration_seconds histogram
rate_limit_service_request_duration_seconds_bucket{le="0.005"} 123
rate_limit_service_request_duration_seconds_bucket{le="0.01"} 456
```

### Health Metrics

```http
GET /health/metrics
```

**Response**
```json
{
  "uptime": "24h",
  "requests": {
    "total": 10000,
    "rate_limited": 150,
    "errors": 10
  },
  "redis": {
    "connected": true,
    "latency_ms": 2,
    "memory_usage": "256MB"
  }
}
```

## Error Responses

### Rate Limit Exceeded

```http
HTTP/1.1 429 Too Many Requests
Content-Type: application/json
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1635724800

{
  "error": "rate_limit_exceeded",
  "message": "Rate limit exceeded for company1",
  "details": {
    "limit": 100,
    "remaining": 0,
    "reset": 1635724800
  }
}
```

### Authentication Error

```http
HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "error": "unauthorized",
  "message": "Invalid or expired JWT token",
  "details": {
    "code": "token_expired"
  }
}
```

## Request/Response Headers

### Request Headers

```http
Authorization: Bearer <jwt-token>
X-Company-ID: company1
X-Request-ID: req-123
X-Forwarded-For: 192.168.1.1
Content-Type: application/json
Accept: application/json
```

### Response Headers

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1635724800
X-Request-ID: req-123
Content-Type: application/json
```

## Rate Limit Headers

### Standard Headers

```http
X-RateLimit-Limit: Maximum requests allowed
X-RateLimit-Remaining: Remaining requests in window
X-RateLimit-Reset: Unix timestamp when limit resets
```

### Debug Headers

```http
X-RateLimit-Debug: true
X-RateLimit-Debug-Info: domain=production;key=company_id;value=company1
```

## WebSocket API

### Connection

```http
GET /ws
```

**Headers**
```http
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: <key>
Sec-WebSocket-Version: 13
```

### Messages

```json
// Subscribe to rate limit events
{
  "type": "subscribe",
  "channel": "rate_limits",
  "filters": {
    "company_id": "company1"
  }
}

// Rate limit event
{
  "type": "event",
  "channel": "rate_limits",
  "data": {
    "company_id": "company1",
    "limit_reached": true,
    "remaining": 0,
    "reset_at": "2024-02-20T10:20:30Z"
  }
}
```

## API Versioning

### Version Header

```http
Accept: application/json; version=1.0
X-API-Version: 1.0
```

### Version URL

```http
GET /v1/users
GET /v2/users
```

## Rate Limit Algorithms

### Fixed Window

```json
{
  "algorithm": "fixed_window",
  "window_size": "60s",
  "limit": 100
}
```

### Sliding Window

```json
{
  "algorithm": "sliding_window",
  "window_size": "60s",
  "limit": 100,
  "precision": "1s"
}
```

### Token Bucket

```json
{
  "algorithm": "token_bucket",
  "capacity": 100,
  "fill_rate": 2,
  "interval": "1s"
}
```

## Next Steps

After reviewing the API:
1. Test API endpoints
2. Implement client libraries
3. Update documentation
4. Monitor API usage 