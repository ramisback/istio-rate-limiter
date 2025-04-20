# Configuration Guide

## Overview

This document details all configuration options for the Istio Rate Limiter project. It covers service configurations, rate limit settings, and infrastructure setup.

## Service Configuration

### 1. Rate Limit Service

#### Environment Variables
```yaml
env:
  # Rate Limiting Configuration
  - name: RATE_LIMIT_WINDOW
    value: "60s"
  - name: IP_RATE_LIMIT
    value: "1000"
  - name: COMPANY_RATE_LIMIT
    value: "10000"
  - name: GLOBAL_RATE_LIMIT
    value: "100000"
  
  # Redis Configuration
  - name: REDIS_CLUSTER_ADDRS
    value: "redis-cluster-0.redis:6379,redis-cluster-1.redis:6379,redis-cluster-2.redis:6379"
  - name: REDIS_PASSWORD
    valueFrom:
      secretKeyRef:
        name: redis-password
        key: password
  
  # Service Configuration
  - name: SERVICE_PORT
    value: "8081"
  - name: METRICS_PORT
    value: "9090"
```

#### Resource Limits
```yaml
resources:
  requests:
    cpu: "100m"
    memory: "256Mi"
  limits:
    cpu: "500m"
    memory: "512Mi"
```

### 2. User Service

#### Environment Variables
```yaml
env:
  # Service Configuration
  - name: SERVICE_PORT
    value: "8080"
  - name: LOG_LEVEL
    value: "info"
  
  # JWT Configuration
  - name: JWT_ISSUER
    value: "issuer.example.com"
  - name: JWT_AUDIENCE
    value: "user-service"
```

#### Resource Limits
```yaml
resources:
  requests:
    cpu: "50m"
    memory: "64Mi"
  limits:
    cpu: "100m"
    memory: "128Mi"
```

## Istio Configuration

### 1. Gateway Configuration
```yaml
apiVersion: networking.istio.io/v1alpha3
kind: Gateway
metadata:
  name: user-service-gateway
spec:
  selector:
    istio: ingressgateway
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    hosts:
    - "*"
```

### 2. Virtual Service
```yaml
apiVersion: networking.istio.io/v1alpha3
kind: VirtualService
metadata:
  name: user-service
spec:
  hosts:
  - "*"
  gateways:
  - user-service-gateway
  http:
  - route:
    - destination:
        host: user-service
        subset: v1
      weight: 100
```

### 3. Destination Rule
```yaml
apiVersion: networking.istio.io/v1alpha3
kind: DestinationRule
metadata:
  name: user-service
spec:
  host: user-service
  subsets:
  - name: v1
    labels:
      version: v1
```

## Rate Limit Configuration

### 1. Rate Limit Filter
```yaml
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: rate-limit-filter
spec:
  workloadSelector:
    labels:
      app: user-service
  configPatches:
    - applyTo: HTTP_FILTER
      match:
        context: SIDECAR_INBOUND
      patch:
        operation: INSERT_BEFORE
        value:
          name: envoy.filters.http.ratelimit
          typed_config:
            "@type": type.googleapis.com/envoy.extensions.filters.http.ratelimit.v3.RateLimit
            domain: rate-limit
            failure_mode_deny: true
            rate_limit_service:
              grpc_service:
                envoy_grpc:
                  cluster_name: rate_limit_cluster
                timeout: 0.25s
```

### 2. JWT Filter
```yaml
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: jwt-filter
spec:
  workloadSelector:
    labels:
      app: user-service
  configPatches:
    - applyTo: HTTP_FILTER
      match:
        context: SIDECAR_INBOUND
      patch:
        operation: INSERT_BEFORE
        value:
          name: envoy.filters.http.jwt_authn
          typed_config:
            "@type": type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication
            providers:
              jwt-provider:
                issuer: "issuer.example.com"
                from_headers:
                  - name: "Authorization"
                    value_prefix: "Bearer "
                remote_jwks:
                  http_uri:
                    uri: "https://issuer.example.com/.well-known/jwks.json"
                    cluster: outbound|80||issuer.example.com
                    timeout: 5s
```

## Redis Configuration

### 1. Redis Cluster
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis-cluster
spec:
  serviceName: redis
  replicas: 3
  template:
    spec:
      containers:
      - name: redis
        image: redis:7.0-alpine
        command: ["redis-server", "/conf/redis.conf"]
        ports:
        - containerPort: 6379
        resources:
          requests:
            cpu: "100m"
            memory: "512Mi"
          limits:
            cpu: "200m"
            memory: "1Gi"
```

### 2. Redis Configuration
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: redis-cluster-config
data:
  redis.conf: |
    cluster-enabled yes
    cluster-config-file /data/nodes.conf
    cluster-node-timeout 5000
    appendonly yes
    maxmemory 1gb
    maxmemory-policy allkeys-lru
```

## Monitoring Configuration

### 1. Prometheus Configuration
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
    scrape_configs:
      - job_name: 'rate-limit-service'
        kubernetes_sd_configs:
          - role: pod
        relabel_configs:
          - source_labels: [__meta_kubernetes_pod_label_app]
            regex: ratelimit
            action: keep
```

### 2. Grafana Dashboard
```json
{
  "dashboard": {
    "title": "Rate Limit Dashboard",
    "panels": [
      {
        "title": "Rate Limit Requests",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(rate_limit_requests_total[5m])",
            "legendFormat": "{{type}}"
          }
        ]
      }
    ]
  }
}
```

## Security Configuration

### 1. Network Policies
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: rate-limit-policy
spec:
  podSelector:
    matchLabels:
      app: ratelimit
  policyTypes:
  - Ingress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: user-service
    ports:
    - protocol: TCP
      port: 8081
```

### 2. Service Account
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: rate-limit-service
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: rate-limit-role
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "watch"]
```

## Environment-Specific Configuration

### 1. Development
```yaml
env:
  - name: LOG_LEVEL
    value: "debug"
  - name: RATE_LIMIT_WINDOW
    value: "10s"
  - name: IP_RATE_LIMIT
    value: "100"
```

### 2. Production
```yaml
env:
  - name: LOG_LEVEL
    value: "info"
  - name: RATE_LIMIT_WINDOW
    value: "60s"
  - name: IP_RATE_LIMIT
    value: "1000"
```

## Configuration Best Practices

### 1. Rate Limit Settings
- Start with conservative limits
- Monitor and adjust based on usage
- Consider different environments
- Use appropriate time windows

### 2. Resource Configuration
- Set appropriate resource limits
- Configure HPA for scaling
- Monitor resource usage
- Plan for capacity

### 3. Security Settings
- Use secure passwords
- Enable mTLS
- Configure network policies
- Implement proper RBAC

### 4. Monitoring Setup
- Enable relevant metrics
- Configure proper retention
- Set up alerting
- Monitor performance 