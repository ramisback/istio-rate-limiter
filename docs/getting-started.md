# Getting Started Guide

This guide provides step-by-step instructions for setting up, configuring, monitoring, and testing the Istio Rate Limiter project.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Installation](#installation)
3. [Configuration](#configuration)
4. [Monitoring](#monitoring)
5. [Load Testing](#load-testing)
6. [Troubleshooting](#troubleshooting)

## Prerequisites

Before starting, ensure you have the following installed:

1. **Kubernetes Cluster**
   Choose one of the following options:
   - Docker Desktop with Kubernetes enabled
   - Minikube
   - Kind
   - Cloud provider's Kubernetes service
   - At least 4 CPUs and 8GB RAM available

2. **Required Tools**

   For Linux:
   ```bash
   # Install kubectl
   curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
   chmod +x kubectl
   sudo mv kubectl /usr/local/bin/

   # Install Helm
   curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash
   ```

   For macOS:
   ```bash
   # Install kubectl using Homebrew
   brew install kubectl

   # Install Helm using Homebrew
   brew install helm
   ```

3. **Clone the Repository**
   ```bash
   git clone https://github.com/yourusername/istio-rate-limiter.git
   cd istio-rate-limiter
   ```

## Installation

### 1. Set Up Kubernetes Cluster

#### Option 1: Using Docker Desktop (Recommended for Local Development)
1. Install Docker Desktop from [https://www.docker.com/products/docker-desktop](https://www.docker.com/products/docker-desktop)
2. Open Docker Desktop
3. Go to Settings/Preferences > Kubernetes
4. Enable Kubernetes
5. Click "Apply & Restart"
6. Wait for Kubernetes to start (the green light in the Docker Desktop status bar)
7. Verify the setup:
   ```bash
   kubectl get nodes
   ```

#### Option 2: Using Minikube
```bash
# Start Minikube
minikube start --cpus 4 --memory 8192 --driver=docker

# Enable ingress addon
minikube addons enable ingress

# Verify cluster is running
kubectl get nodes
```

#### Option 3: Using Cloud Provider (Production)
Follow your cloud provider's instructions to create a Kubernetes cluster with:
- At least 4 CPUs
- 8GB RAM
- Ingress controller enabled

### 2. Install Istio

Istio should be installed in your Kubernetes cluster, not on your local machine. The `istioctl` CLI tool is only needed for installation and management.

#### Option 1: Using istioctl (Recommended)
```bash
# Download istioctl
curl -L https://istio.io/downloadIstio | sh -
cd istio-*
export PATH=$PWD/bin:$PATH

# Install Istio with the demo profile
istioctl install --set profile=demo -y

# Verify installation
istioctl verify-install
```

#### Option 2: Using Helm
```bash
# Add Istio Helm repository
helm repo add istio https://istio-release.storage.googleapis.com/charts
helm repo update

# Install Istio base
helm install istio-base istio/base -n istio-system --create-namespace

# Install Istio demo profile
helm install istio-demo istio/demo -n istio-system
```

### 3. Deploy the Project Components

```bash
# Create namespace
kubectl create namespace istio-rate-limiter

# Deploy Redis
kubectl apply -f k8s/redis-cluster.yaml

# Deploy Rate Limit Service
kubectl apply -f k8s/ratelimit-deployment.yaml

# Deploy User Service
kubectl apply -f k8s/user-service.yaml

# Deploy Istio Gateway and Virtual Service
kubectl apply -f k8s/gateway.yaml
kubectl apply -f k8s/virtual-service.yaml

# Deploy Rate Limit Filter
kubectl apply -f k8s/ratelimit-filter.yaml

# Deploy JWT Filter
kubectl apply -f k8s/jwt-filter.yaml

# Deploy Monitoring Stack
kubectl apply -f k8s/prometheus-config.yaml
kubectl apply -f k8s/grafana-datasource.yaml
kubectl apply -f k8s/grafana-dashboard.yaml
```

### 4. Verify Deployment

```bash
# Check all pods are running
kubectl get pods -n istio-rate-limiter

# Check services
kubectl get svc -n istio-rate-limiter

# Check Istio resources
kubectl get gateway,virtualservice,envoyfilter -n istio-rate-limiter

# Get the external IP or set up port forwarding
if [[ $(kubectl config current-context) == *"docker-desktop"* ]]; then
    # For Docker Desktop
    kubectl port-forward -n istio-system svc/istio-ingressgateway 8080:80 &
    echo "Access the service at http://localhost:8080"
elif [[ $(minikube status -o json | jq -r '.Host') == "Running" ]]; then
    # For Minikube
    minikube service istio-ingressgateway -n istio-system --url
else
    # For cloud providers
    kubectl get svc -n istio-system istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
fi
```

## Configuration

### 1. Rate Limit Configuration

The rate limits are configured in the `k8s/ratelimit-filter.yaml` file:

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

To modify rate limits:

1. **IP-based Rate Limiting**
   - Edit `rate-limit-service/main.go`
   - Find the `ipLimit` variable
   - Change the value (default: 1000 requests per minute)

2. **Company-based Rate Limiting**
   - Edit `rate-limit-service/main.go`
   - Find the `companyLimit` variable
   - Change the value (default: 10000 requests per minute)

3. **Global Rate Limiting**
   - Edit `rate-limit-service/main.go`
   - Find the `globalLimit` variable
   - Change the value (default: 100000 requests per minute)

After changing limits, redeploy the rate limit service:

```bash
kubectl rollout restart deployment ratelimit -n istio-rate-limiter
```

### 2. JWT Configuration

JWT authentication is configured in `k8s/jwt-filter.yaml`:

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

To modify JWT settings:

1. **JWT Issuer**
   - Edit `user-service/internal/service/auth.go`
   - Find the `issuer` variable
   - Change to your issuer

2. **JWT Expiry**
   - Edit `user-service/internal/service/auth.go`
   - Find the `expiry` variable
   - Change the duration (default: 1 hour)

After changing JWT settings, redeploy the user service:

```bash
kubectl rollout restart deployment user-service -n istio-rate-limiter
```

## Monitoring

### 1. Access Grafana Dashboard

```bash
# Port forward Grafana
kubectl port-forward -n istio-rate-limiter svc/grafana 3000:3000

# Access in browser
open http://localhost:3000
# Default credentials: admin/admin
```

### 2. Access Prometheus

```bash
# Port forward Prometheus
kubectl port-forward -n istio-rate-limiter svc/prometheus 9090:9090

# Access in browser
open http://localhost:9090
```

### 3. Key Metrics to Monitor

1. **Rate Limit Metrics**
   ```
   # Rate limit requests
   rate(rate_limit_requests_total[5m])
   
   # Rate limit hits
   rate(rate_limit_hits_total[5m])
   
   # Rate limit latency
   histogram_quantile(0.95, rate(rate_limit_latency_seconds_bucket[5m]))
   ```

2. **User Service Metrics**
   ```
   # Request rate
   rate(user_service_requests_total[5m])
   
   # Error rate
   rate(user_service_errors_total[5m])
   
   # Latency
   histogram_quantile(0.95, rate(user_service_latency_seconds_bucket[5m]))
   ```

3. **Redis Metrics**
   ```
   # Redis operations
   rate(redis_operations_total[5m])
   
   # Redis latency
   histogram_quantile(0.95, rate(redis_latency_seconds_bucket[5m]))
   ```

### 4. Grafana Dashboards

1. **Rate Limiter Dashboard**
   - Shows rate limit requests, hits, and latency
   - Located at: `http://localhost:3000/d/rate-limiter/rate-limiter`

2. **User Service Dashboard**
   - Shows user service metrics
   - Located at: `http://localhost:3000/d/user-service/user-service`

3. **Redis Dashboard**
   - Shows Redis performance metrics
   - Located at: `http://localhost:3000/d/redis/redis`

## Load Testing

### 1. Using the Helper Script

The easiest way to run load tests is using the provided helper script:

```bash
# Make the script executable
chmod +x run-loadtest.sh

# Run the load test
./run-loadtest.sh
```

This script will:
1. Detect the external IP of the Istio ingress gateway
2. Set up port forwarding if needed
3. Build and run the load test
4. Clean up resources when done

### 2. Manual Load Testing

If you prefer to run tests manually:

```bash
# Get the Gateway IP
EXTERNAL_IP=$(kubectl get svc -n istio-system istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Set up Port Forwarding (if needed)
kubectl port-forward -n istio-system svc/istio-ingressgateway 8080:80 &
PF_PID=$!
EXTERNAL_IP="http://localhost:8080"

# Build and Run
cd loadtest
go build -o loadtest
./loadtest -url "$EXTERNAL_IP" -rps 100 -duration 5m -concurrency 10

# Clean up port forwarding if needed
if [ ! -z "$PF_PID" ]; then
  kill $PF_PID
fi
```

### 3. Test Scenarios

1. **Basic Rate Limiting**
   ```bash
   ./loadtest -url "$EXTERNAL_IP" -rps 50 -duration 1m
   ```

2. **Authentication Flow**
   ```bash
   ./loadtest -url "$EXTERNAL_IP" -rps 30 -duration 2m -auth
   ```

3. **High Load**
   ```bash
   ./loadtest -url "$EXTERNAL_IP" -rps 200 -duration 3m -concurrency 20
   ```

4. **Error Handling**
   ```bash
   ./loadtest -url "$EXTERNAL_IP" -rps 100 -duration 1m -error-rate 0.1
   ```

### 4. Analyzing Results

After running load tests, check the following:

1. **Grafana Dashboard**
   - Open `http://localhost:3000/d/load-test/load-test`
   - Review request rates, latencies, and error rates

2. **Prometheus Queries**
   ```
   # Request success rate
   sum(rate(loadtest_requests_total{status="success"}[5m])) / sum(rate(loadtest_requests_total[5m]))
   
   # Rate limit hits
   sum(rate(loadtest_rate_limits_hit_total[5m]))
   
   # Error rate
   sum(rate(loadtest_requests_total{status="error"}[5m])) / sum(rate(loadtest_requests_total[5m]))
   ```

## Troubleshooting

### 1. Common Issues

1. **Gateway Not Accessible**
   ```bash
   # Check Istio ingress gateway status
   kubectl get svc -n istio-system istio-ingressgateway
   
   # Check gateway logs
   kubectl logs -n istio-system -l app=istio-ingressgateway
   ```

2. **Rate Limits Not Applied**
   ```bash
   # Check rate limit service logs
   kubectl logs -l app=ratelimit
   
   # Check Envoy configuration
   istioctl proxy-config listener -n istio-system $(kubectl get pod -n istio-system -l app=istio-ingressgateway -o jsonpath='{.items[0].metadata.name}')
   ```

3. **High Error Rates**
   ```bash
   # Check user service logs
   kubectl logs -l app=user-service
   
   # Check Redis connection
   kubectl exec -it $(kubectl get pod -l app=redis -o jsonpath='{.items[0].metadata.name}') -- redis-cli ping
   ```

### 2. Debugging Tools

1. **Service Logs**
   ```bash
   kubectl logs -l app=user-service
   kubectl logs -l app=ratelimit
   ```

2. **Metrics**
   ```bash
   kubectl port-forward svc/prometheus 9090:9090
   ```

3. **Configuration**
   ```bash
   kubectl get gateway,virtualservice,envoyfilter
   ```

### 3. Recovery Procedures

1. **Rate Limit Service**
   ```bash
   # Scale down service
   kubectl scale deployment ratelimit --replicas=0
   
   # Clear Redis cache
   kubectl exec -it $(kubectl get pod -l app=redis -o jsonpath='{.items[0].metadata.name}') -- redis-cli FLUSHALL
   
   # Scale up service
   kubectl scale deployment ratelimit --replicas=3
   ```

2. **User Service**
   ```bash
   # Restart service
   kubectl rollout restart deployment user-service
   
   # Verify service
   kubectl rollout status deployment user-service
   ```

## Next Steps

After getting started with the basic setup, you can:

1. **Customize Rate Limits**
   - Adjust limits for different types of requests
   - Implement custom rate limiting strategies

2. **Enhance Monitoring**
   - Create custom Grafana dashboards
   - Set up alerts for rate limit violations

3. **Scale the System**
   - Increase Redis cluster size
   - Add more rate limit service replicas
   - Configure horizontal pod autoscaling

4. **Implement Advanced Features**
   - Add circuit breakers
   - Implement retry policies
   - Configure traffic shifting

For more detailed information, refer to the specific documentation files:
- [Architecture](01-architecture.md)
- [Rate Limiting](04-rate-limiting.md)
- [Configuration](05-configuration.md)
- [Load Testing](06-load-testing.md)
- [Monitoring](07-monitoring.md)
- [Troubleshooting](08-troubleshooting.md) 