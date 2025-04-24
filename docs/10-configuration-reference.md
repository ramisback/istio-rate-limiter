# Configuration Reference

This document provides a detailed explanation of each configuration file in the project.

## Kubernetes Configuration Files

### Gateway Configuration (`k8s/gateway.yaml`)
```yaml
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: user-gateway
  namespace: default
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
This file defines the Istio Gateway that acts as the entry point for external traffic. It:
- Listens on port 80 for HTTP traffic
- Accepts traffic from any host (`"*"`)
- Uses the default Istio ingress gateway

### Virtual Service (`k8s/virtual-service.yaml`)
```yaml
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: user-vs
  namespace: default
spec:
  hosts:
  - "*"
  gateways:
  - user-gateway
  http:
  - match:
    - uri:
        prefix: "/api"
    route:
    - destination:
        host: user-service
        port:
          number: 8083
```
This file defines how traffic is routed within the service mesh:
- Routes all traffic from the gateway to the user service
- Matches requests with the `/api` prefix
- Forwards traffic to port 8083 of the user service

### Destination Rule (`k8s/destination-rule.yaml`)
```yaml
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: user-destination
  namespace: default
spec:
  host: user-service
  trafficPolicy:
    loadBalancer:
      simple: ROUND_ROBIN
```
This file defines how traffic is distributed to the user service:
- Uses round-robin load balancing
- Applies to all traffic destined for the user service

### Rate Limit Filter (`k8s/ratelimit-filter.yaml`)
```yaml
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: filter-ratelimit
  namespace: default
spec:
  workloadSelector:
    labels:
      istio: ingressgateway
  configPatches:
  - applyTo: HTTP_FILTER
    match:
      context: GATEWAY
      listener:
        filterChain:
          filter:
            name: "envoy.filters.network.http_connection_manager"
    patch:
      operation: MERGE
      value:
        name: envoy.filters.http.ratelimit
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.http.ratelimit.v3.RateLimit
          domain: user
          rate_limit_service:
            grpc_service:
              envoy_grpc:
                cluster_name: rate_limit_cluster
            timeout: 0.25s
```
This file configures the rate limiting filter:
- Applies to the ingress gateway
- Uses the rate limit service for rate limiting
- Sets a 250ms timeout for rate limit checks
- Uses the "user" domain for rate limiting

### Rate Limit Deployment (`k8s/ratelimit-deployment.yaml`)
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ratelimit
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ratelimit
  template:
    metadata:
      labels:
        app: ratelimit
    spec:
      containers:
      - name: ratelimit
        image: envoyproxy/ratelimit:latest
        ports:
        - containerPort: 8081
        env:
        - name: REDIS_URL
          value: "redis:6379"
        - name: LOG_LEVEL
          value: debug
```
This file defines the rate limit service deployment:
- Uses the official Envoy rate limit image
- Connects to Redis for rate limit storage
- Exposes port 8081 for gRPC communication

### Redis Cluster (`k8s/redis-cluster.yaml`)
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: redis
  namespace: default
spec:
  serviceName: redis
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:6.2-alpine
        ports:
        - containerPort: 6379
```
This file defines the Redis deployment:
- Uses Redis 6.2 Alpine for rate limit storage
- Runs as a StatefulSet for data persistence
- Exposes port 6379 for Redis communication

### JWT Filter (`k8s/jwt-filter.yaml`)
```yaml
apiVersion: networking.istio.io/v1alpha3
kind: EnvoyFilter
metadata:
  name: jwt-filter
  namespace: default
spec:
  workloadSelector:
    labels:
      istio: ingressgateway
  configPatches:
  - applyTo: HTTP_FILTER
    match:
      context: GATEWAY
      listener:
        filterChain:
          filter:
            name: "envoy.filters.network.http_connection_manager"
    patch:
      operation: MERGE
      value:
        name: envoy.filters.http.jwt_authn
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.http.jwt_authn.v3.JwtAuthentication
```
This file configures JWT authentication:
- Applies to the ingress gateway
- Validates JWT tokens for authenticated endpoints
- Integrates with the rate limiting system

## Load Test Configuration

### Load Test Script (`run-loadtest.sh`)
```bash
#!/bin/bash
# Get the external IP of the Istio ingress gateway
EXTERNAL_IP=$(kubectl get svc -n istio-system istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

if [ -z "$EXTERNAL_IP" ]; then
  # Fallback to port forwarding if external IP is not available
  kubectl port-forward -n istio-system svc/istio-ingressgateway 8080:80 &
  PF_PID=$!
  EXTERNAL_IP="http://localhost:8080"
  sleep 5
else
  EXTERNAL_IP="http://$EXTERNAL_IP"
fi

# Build and run the load test
cd loadtest
go build -o loadtest
./loadtest -url "$EXTERNAL_IP" -rps 100 -duration 5m -concurrency 10

# Clean up port forwarding if needed
if [ ! -z "$PF_PID" ]; then
  kill $PF_PID
fi
```
This script:
- Automatically detects the external IP of the Istio ingress gateway
- Falls back to port forwarding if needed
- Builds and runs the load test with default parameters
- Cleans up port forwarding when done

### Load Test Configuration (`loadtest/main.go`)
The load test is configured with the following parameters:
- `-url`: Target URL (default: http://localhost:8083)
- `-rps`: Requests per second (default: 100)
- `-duration`: Test duration (default: 5m)
- `-concurrency`: Number of concurrent workers (default: 10)
- `-metrics`: Enable Prometheus metrics (default: true)
- `-metrics-port`: Metrics port (default: 9090)

## Monitoring Configuration

### Prometheus Config (`k8s/prometheus-config.yaml`)
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: default
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
    scrape_configs:
      - job_name: 'istio'
        static_configs:
          - targets: ['istio-pilot.istio-system:9090']
      - job_name: 'user-service'
        static_configs:
          - targets: ['user-service:8083']
```
This file configures Prometheus monitoring:
- Sets a 15-second scrape interval
- Monitors Istio control plane metrics
- Collects metrics from the user service

### Grafana Dashboard (`k8s/grafana-dashboard.yaml`)
This file defines a Grafana dashboard for visualizing:
- Request rates and latencies
- Rate limit hits and rejections
- Service health metrics
- Resource utilization

### Grafana Datasource (`k8s/grafana-datasource.yaml`)
This file configures the Prometheus data source for Grafana:
- Connects to the Prometheus service
- Enables metric visualization
- Sets up default queries and variables 