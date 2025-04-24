# Load Test for Istio Rate Limiter Demo

This load test tool is designed to generate traffic to the user service to demonstrate the rate limiting capabilities of Istio.

## Running the Load Test Locally

You can run the load test from your local machine to test the rate limiting functionality. The load test will connect to the user service through the Istio ingress gateway.

### Prerequisites

- Kubernetes cluster with Istio installed
- `kubectl` configured to access your cluster
- Go 1.16 or later

### Option 1: Using the Helper Script

The easiest way to run the load test is to use the helper script:

```bash
# From the root directory of the project
./run-loadtest.sh
```

This script will:
1. Get the external IP of the Istio ingress gateway
2. Set up port forwarding if needed
3. Build and run the load test

### Option 2: Manual Setup

If you prefer to run the load test manually:

1. Get the external IP of the Istio ingress gateway:
   ```bash
   # If using a cloud provider with LoadBalancer
   EXTERNAL_IP=$(kubectl get svc -n istio-system istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
   
   # Or set up port forwarding
   kubectl port-forward -n istio-system svc/istio-ingressgateway 8080:80
   ```

2. Build the load test:
   ```bash
   cd loadtest
   go build -o loadtest
   ```

3. Run the load test:
   ```bash
   # If using LoadBalancer
   ./loadtest -url "http://$EXTERNAL_IP" -rps 100 -duration 5m -concurrency 10
   
   # If using port forwarding
   ./loadtest -url "http://localhost:8080" -rps 100 -duration 5m -concurrency 10
   ```

## Load Test Parameters

- `-url`: Target URL (default: http://localhost:8083)
- `-rps`: Requests per second (default: 100)
- `-duration`: Test duration (default: 5m)
- `-concurrency`: Number of concurrent workers (default: 10)
- `-metrics`: Enable Prometheus metrics (default: true)
- `-metrics-port`: Metrics port (default: 9090)

## Endpoints

The load test will randomly select from the following endpoints:

- `/fast`: 10ms response time
- `/medium`: 100ms response time
- `/slow`: 500ms response time
- `/very-slow`: 1s response time

## Monitoring

The load test exposes Prometheus metrics at `:9090/metrics` that can be used to monitor the test results.

## Rate Limiting

The rate limiting is configured in the `ratelimit-filter.yaml` file and applies different rate limits based on the endpoint path. You can adjust the rate limits by modifying the Redis configuration in the `ratelimit-deployment.yaml` file. 