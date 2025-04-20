# Installation Guide

## Overview

This guide provides step-by-step instructions for installing and setting up the Istio Rate Limiter project. Follow each section in sequence to ensure a proper setup.

## Prerequisites Check

Before proceeding, ensure all prerequisites are met:

```bash
# Verify required tools
docker --version        # Should be 4.x or later
kubectl version        # Should be 1.24 or later
istioctl version      # Should be 1.18 or later
go version            # Should be 1.21 or later
redis-cli --version   # Should be 7.x or later
```

## Project Setup

### 1. Clone Repository
```bash
# Clone the repository
git clone git@github.com:ramisback/istio-rate-limiter.git
cd istio-rate-limiter

# Check out the latest stable version
git checkout v1.0.0  # or latest tag
```

### 2. Environment Configuration
```bash
# Set required environment variables
export ISTIO_VERSION=1.18.0
export REDIS_VERSION=7.0
export GO_VERSION=1.21

# Configure local environment
source ./scripts/setup-env.sh  # if available
```

## Infrastructure Setup

### 1. Kubernetes Cluster
```bash
# If using Docker Desktop:
# Enable Kubernetes in Docker Desktop settings

# If using kind:
kind create cluster --name istio-ratelimit --config k8s/kind-config.yaml

# Verify cluster is ready
kubectl cluster-info
kubectl get nodes
```

### 2. Istio Installation
```bash
# Download Istio
curl -L https://istio.io/downloadIstio | sh -
cd istio-*
export PATH=$PWD/bin:$PATH

# Install Istio
istioctl install --set profile=demo -y

# Enable injection for namespaces
kubectl label namespace default istio-injection=enabled
kubectl create namespace redis
kubectl label namespace redis istio-injection=enabled

# Verify Istio installation
kubectl get pods -n istio-system
```

### 3. Redis Cluster Setup
```bash
# Deploy Redis cluster
kubectl apply -f k8s/redis/

# Wait for Redis pods
kubectl wait --for=condition=Ready pods -l app=redis-cluster -n redis

# Initialize Redis cluster
kubectl exec -it redis-cluster-0 -n redis -- redis-cli --cluster create \
  $(kubectl get pods -l app=redis-cluster -n redis -o jsonpath='{range.items[*]}{.status.podIP}:6379 {end}') \
  --cluster-replicas 1

# Verify Redis cluster
kubectl exec -it redis-cluster-0 -n redis -- redis-cli cluster info
```

## Service Deployment

### 1. Build Services
```bash
# Build rate limit service
cd rate-limit-service
docker build -t ratelimit:v1 .
cd ..

# Build user service
cd user-service
docker build -t user-service:v1 .
cd ..

# If using kind, load images
kind load docker-image ratelimit:v1 --name istio-ratelimit
kind load docker-image user-service:v1 --name istio-ratelimit
```

### 2. Deploy Services
```bash
# Deploy rate limit service
kubectl apply -f k8s/ratelimit/
kubectl wait --for=condition=Ready pods -l app=ratelimit

# Deploy user service
kubectl apply -f k8s/user-service/
kubectl wait --for=condition=Ready pods -l app=user-service
```

### 3. Configure Istio Resources
```bash
# Apply Istio configurations
kubectl apply -f k8s/istio/gateway.yaml
kubectl apply -f k8s/istio/virtual-service.yaml
kubectl apply -f k8s/istio/destination-rule.yaml
kubectl apply -f k8s/istio/jwt-filter.yaml
kubectl apply -f k8s/istio/ratelimit-filter.yaml

# Verify Istio resources
kubectl get gateway
kubectl get virtualservice
kubectl get destinationrule
kubectl get envoyfilter
```

## Monitoring Setup (Optional)

### 1. Deploy Prometheus
```bash
# Create monitoring namespace
kubectl create namespace monitoring
kubectl label namespace monitoring istio-injection=enabled

# Deploy Prometheus
kubectl apply -f monitoring/prometheus.yaml
```

### 2. Deploy Grafana
```bash
# Deploy Grafana
kubectl apply -f monitoring/grafana.yaml

# Import dashboards
kubectl apply -f monitoring/dashboards/
```

## Verification

### 1. Check Service Status
```bash
# Check all pods are running
kubectl get pods -A | grep -E 'ratelimit|user-service|redis|prometheus|grafana'

# Check all services
kubectl get svc -A | grep -E 'ratelimit|user-service|redis|prometheus|grafana'
```

### 2. Test Services
```bash
# Get Istio ingress gateway URL
export INGRESS_HOST=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
export INGRESS_PORT=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.spec.ports[?(@.name=="http2")].port}')
export GATEWAY_URL=$INGRESS_HOST:$INGRESS_PORT

# Test health endpoint
curl -I http://$GATEWAY_URL/health

# Test with JWT token
TOKEN=$(curl -s http://$GATEWAY_URL/auth/token)
curl -H "Authorization: Bearer $TOKEN" http://$GATEWAY_URL/users
```

### 3. Verify Rate Limiting
```bash
# Test IP-based rate limiting
for i in {1..50}; do
  curl -I http://$GATEWAY_URL/users
  sleep 0.1
done

# Test company-based rate limiting
for i in {1..20}; do
  curl -H "Authorization: Bearer $TOKEN" http://$GATEWAY_URL/users
  sleep 0.1
done
```

## Post-Installation

### 1. Access Dashboards
```bash
# Access Grafana
kubectl port-forward -n monitoring svc/grafana 3000:3000

# Access Prometheus
kubectl port-forward -n monitoring svc/prometheus 9090:9090
```

### 2. Configure Logging
```bash
# Set Envoy logging level
kubectl exec -it $(kubectl get pod -l app=user-service -o jsonpath='{.items[0].metadata.name}') -c istio-proxy -- curl -X POST localhost:15000/logging?level=debug
```

### 3. Setup Alerts (Optional)
```bash
# Apply alert rules
kubectl apply -f monitoring/alerts/

# Configure alert manager
kubectl apply -f monitoring/alertmanager/
```

## Troubleshooting

### Common Issues

1. **Pods not starting:**
   ```bash
   kubectl describe pod <pod-name>
   kubectl logs <pod-name>
   ```

2. **Rate limiting not working:**
   ```bash
   # Check rate limit service logs
   kubectl logs -l app=ratelimit
   
   # Check Envoy config
   istioctl proxy-config all <pod-name>
   ```

3. **Redis cluster issues:**
   ```bash
   # Check Redis cluster status
   kubectl exec -it redis-cluster-0 -n redis -- redis-cli cluster info
   ```

### Cleanup

To remove the installation:
```bash
# Delete all resources
kubectl delete -f k8s/
kubectl delete namespace redis
kubectl delete namespace monitoring

# Remove Istio
istioctl x uninstall --purge

# Delete cluster (if using kind)
kind delete cluster --name istio-ratelimit
``` 