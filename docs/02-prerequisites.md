# Prerequisites

## Required Tools and Versions

### 1. Docker Desktop
- Version: 4.x or later
- Features:
  - Kubernetes enabled
  - At least 4GB memory allocated
  - Linux containers mode

### 2. Kubernetes
- Version: 1.24 or later
- Components:
  - kubectl CLI tool
  - Metrics server
  - RBAC enabled
  - Storage class configured

### 3. Istio
- Version: 1.18 or later
- Components:
  - istioctl CLI
  - Istio core
  - Istio ingress gateway
  - Istio egress gateway (optional)

### 4. Redis
- Version: 7.x or later
- Mode: Cluster
- Requirements:
  - Minimum 3 master nodes
  - Minimum 3 replica nodes
  - Persistent storage

### 5. Development Tools
- Go 1.24.2 or later
- Git
- Make
- curl
- jq (JSON processor)
- Docker Compose (optional)

## System Requirements

### Minimum Hardware
- CPU: 4 cores
- Memory: 8GB RAM
- Storage: 20GB free space
- Network: Stable internet connection

### Recommended Hardware
- CPU: 8 cores
- Memory: 16GB RAM
- Storage: 40GB free space
- Network: High-speed internet connection

## Environment Setup

### 1. Verify Tool Installation
```bash
# Check tool versions
docker --version
kubectl version --short
istioctl version
go version
redis-cli --version
git --version
make --version
curl --version
jq --version
```

### 2. Configure Docker Desktop
1. Open Docker Desktop preferences
2. Enable Kubernetes
3. Allocate resources:
   ```
   Memory: 8GB minimum
   CPU: 4 cores minimum
   Swap: 2GB
   Disk image size: 40GB
   ```
4. Apply and restart

## Network Requirements

### 1. Kubernetes Networking
- Pod network CIDR configured
- Service network CIDR configured
- NodePort or LoadBalancer access
- Network policies enabled

### 2. Firewall Rules
- Allow Kubernetes API server ports
- Allow Istio ingress gateway ports
- Allow Redis cluster ports
- Allow monitoring ports (optional)

### 3. DNS Configuration
- Kubernetes DNS service running
- External DNS resolution working
- Custom domains configured (if needed)

## Security Requirements

### 1. Kubernetes Security
- RBAC enabled
- Network policies configured
- Pod security policies (optional)
- Service accounts configured

### 2. Istio Security
- mTLS enabled
- JWT validation configured
- Authorization policies set
- Security policies defined

### 3. Redis Security
- Password authentication
- TLS encryption (optional)
- Network isolation
- Access controls

## Monitoring Setup (Optional)

### 1. Prometheus
- Version: 2.x or later
- Storage configured
- Service discovery enabled
- Alert rules defined

### 2. Grafana
- Version: 8.x or later
- Dashboards imported
- Data sources configured
- User access configured

## Development Environment

### 1. IDE/Editor Setup
- Go plugins installed
- Kubernetes support
- YAML validation
- Code formatting

### 2. Local Tools
- Docker Compose (for local testing)
- Skaffold (for development)
- Helm (for package management)
- kubectx/kubens (for context switching)

## Verification Checklist

### 1. Kubernetes Cluster
- [ ] Kubernetes running and accessible
- [ ] kubectl configured correctly
- [ ] Nodes ready and healthy
- [ ] Storage class available

### 2. Istio Installation
- [ ] Istio pods running
- [ ] Ingress gateway configured
- [ ] Injection working
- [ ] mTLS enabled

### 3. Redis Cluster
- [ ] Redis pods running
- [ ] Cluster initialized
- [ ] Persistence working
- [ ] Connectivity verified

### 4. Development Tools
- [ ] All required tools installed
- [ ] Correct versions verified
- [ ] Environment variables set
- [ ] Access permissions configured

## Environment Variables

### Required Variables
```bash
# Istio Gateway URL
export GATEWAY_URL=$(kubectl -n istio-system get service istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Redis connection
export REDIS_HOST=redis-cluster.redis.svc.cluster.local
export REDIS_PORT=6379

# JWT settings
export JWT_SECRET=your-secret-key
export JWT_ISSUER=your-issuer
```

### Optional Variables
```bash
# Debug mode
export DEBUG=true

# Custom ports
export SERVICE_PORT=8080
export METRICS_PORT=9090

# Rate limit settings
export DEFAULT_REQUESTS_PER_MINUTE=100
export IP_REQUESTS_PER_MINUTE=30
```

## Next Steps

After completing the prerequisites:
1. Verify all tools are installed and working
2. Proceed to the [Installation Guide](03-installation.md) 