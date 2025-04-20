# Troubleshooting Guide

## Overview

This guide helps diagnose and resolve common issues in the rate-limited service:
- Installation issues
- Configuration problems
- Runtime errors
- Performance issues
- Common failure scenarios

## Diagnostic Tools

### 1. Kubernetes Tools

```bash
# Check pod status
kubectl get pods -A | grep -E 'ratelimit|user-service|redis'

# Check pod logs
kubectl logs -l app=ratelimit
kubectl logs -l app=user-service

# Describe resources
kubectl describe pod <pod-name>
kubectl describe svc <service-name>
```

### 2. Istio Tools

```bash
# Check Istio proxy status
istioctl proxy-status

# Analyze Istio configuration
istioctl analyze

# Debug Envoy configuration
istioctl proxy-config all <pod-name>
```

## Common Issues

### 1. Installation Issues

#### Pod Not Starting
```bash
# Symptom: Pods stuck in Pending state
kubectl get pods
kubectl describe pod <pod-name>

# Solution:
# 1. Check resource constraints
kubectl describe node | grep -A 5 "Allocated resources"

# 2. Check PVC status
kubectl get pvc
kubectl describe pvc <pvc-name>

# 3. Check node affinity
kubectl get pods -o wide
```

#### Service Not Accessible
```bash
# Symptom: Service endpoints not reachable
kubectl get endpoints
kubectl get svc

# Solution:
# 1. Check service selectors
kubectl describe svc <service-name>

# 2. Verify pod labels
kubectl get pods --show-labels

# 3. Test service DNS
kubectl run -it --rm debug --image=busybox -- nslookup <service-name>
```

### 2. Rate Limiting Issues

#### Rate Limits Not Applied
```bash
# Symptom: Requests not being rate limited
# 1. Check Envoy configuration
istioctl proxy-config listener <pod-name>

# 2. Verify rate limit service logs
kubectl logs -l app=ratelimit

# 3. Check Redis connection
kubectl exec -it redis-cluster-0 -- redis-cli ping
```

#### Incorrect Rate Limits
```bash
# Symptom: Wrong limits being applied
# 1. Check rate limit configuration
kubectl get envoyfilter
kubectl describe envoyfilter <filter-name>

# 2. Verify headers
curl -v -H "X-Company-ID: company1" http://$GATEWAY_URL/users

# 3. Check rate limit counters
kubectl exec -it redis-cluster-0 -- redis-cli keys "rate_limit:*"
```

### 3. Redis Issues

#### Redis Cluster Problems
```bash
# Symptom: Redis cluster unhealthy
# 1. Check cluster status
kubectl exec -it redis-cluster-0 -- redis-cli cluster info

# 2. Check node status
kubectl exec -it redis-cluster-0 -- redis-cli cluster nodes

# 3. Monitor Redis metrics
kubectl port-forward svc/redis-metrics 9121:9121
curl localhost:9121/metrics
```

#### Redis Performance Issues
```bash
# Symptom: High Redis latency
# 1. Check Redis info
kubectl exec -it redis-cluster-0 -- redis-cli info

# 2. Monitor slow logs
kubectl exec -it redis-cluster-0 -- redis-cli slowlog get 10

# 3. Check memory usage
kubectl exec -it redis-cluster-0 -- redis-cli info memory
```

### 4. JWT Authentication Issues

#### Token Validation Failures
```bash
# Symptom: JWT validation errors
# 1. Check JWT configuration
kubectl get RequestAuthentication
kubectl describe RequestAuthentication

# 2. Verify token
echo "<token>" | jwt decode -

# 3. Check Envoy JWT filter
istioctl proxy-config all -n default <pod-name> | grep jwt
```

#### Token Generation Issues
```bash
# Symptom: Invalid tokens generated
# 1. Check token service logs
kubectl logs -l app=user-service -c token-service

# 2. Verify token claims
curl -v http://$GATEWAY_URL/auth/token | jq

# 3. Test token validation
curl -H "Authorization: Bearer <token>" http://$GATEWAY_URL/users
```

## Performance Issues

### 1. High Latency

#### Service Latency
```bash
# Symptom: High request latency
# 1. Check service metrics
curl -s http://$GATEWAY_URL/metrics | grep request_duration

# 2. Profile service
kubectl exec -it <pod-name> -- go tool pprof http://localhost:6060/debug/pprof/profile

# 3. Check resource usage
kubectl top pod <pod-name>
```

#### Network Latency
```bash
# Symptom: Network delays
# 1. Test network latency
kubectl run -it --rm nettools --image=nicolaka/netshoot -- ping <service-name>

# 2. Check DNS resolution
kubectl run -it --rm nettools --image=nicolaka/netshoot -- dig <service-name>

# 3. Monitor network metrics
kubectl top pod <pod-name> --containers
```

### 2. Resource Exhaustion

#### CPU Usage
```bash
# Symptom: High CPU utilization
# 1. Check CPU metrics
kubectl top pods
kubectl describe pod <pod-name>

# 2. Analyze CPU profile
kubectl exec -it <pod-name> -- go tool pprof http://localhost:6060/debug/pprof/profile

# 3. Review resource limits
kubectl get pod <pod-name> -o yaml | grep resources -A 8
```

#### Memory Usage
```bash
# Symptom: High memory usage
# 1. Check memory metrics
kubectl top pods
kubectl describe pod <pod-name>

# 2. Analyze memory profile
kubectl exec -it <pod-name> -- go tool pprof http://localhost:6060/debug/pprof/heap

# 3. Check for memory leaks
kubectl logs <pod-name> | grep "memory"
```

## Recovery Procedures

### 1. Service Recovery

#### Rate Limit Service
```bash
# 1. Scale down service
kubectl scale deployment rate-limit --replicas=0

# 2. Clear Redis cache
kubectl exec -it redis-cluster-0 -- redis-cli FLUSHALL

# 3. Scale up service
kubectl scale deployment rate-limit --replicas=3

# 4. Verify recovery
kubectl get pods -l app=rate-limit
curl -v http://$GATEWAY_URL/health
```

#### User Service
```bash
# 1. Backup configuration
kubectl get configmap user-service-config -o yaml > backup.yaml

# 2. Restart service
kubectl rollout restart deployment user-service

# 3. Verify service
kubectl rollout status deployment user-service
curl -v http://$GATEWAY_URL/health
```

### 2. Data Recovery

#### Redis Backup
```bash
# 1. Create backup
kubectl exec -it redis-cluster-0 -- redis-cli SAVE

# 2. Copy backup file
kubectl cp redis-cluster-0:/data/dump.rdb backup.rdb

# 3. Restore if needed
kubectl cp backup.rdb redis-cluster-0:/data/dump.rdb
kubectl exec -it redis-cluster-0 -- redis-cli BGREWRITEAOF
```

#### Configuration Backup
```bash
# 1. Backup all configs
kubectl get configmap,secret -l app=rate-limit -o yaml > configs-backup.yaml

# 2. Restore configs if needed
kubectl apply -f configs-backup.yaml

# 3. Verify configuration
kubectl get configmap,secret -l app=rate-limit
```

## Preventive Measures

### 1. Health Checks

```yaml
# Pod health check configuration
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 15
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

### 2. Monitoring Alerts

```yaml
# Prometheus alert rules
groups:
- name: rate-limit-alerts
  rules:
  - alert: ServiceDown
    expr: up{job="rate-limit"} == 0
    for: 5m
    labels:
      severity: critical
  - alert: HighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
```

## Next Steps

After troubleshooting:
1. Document the issue and solution
2. Update monitoring if needed
3. Review and update alerts
4. Proceed to [API Reference](09-api-reference.md) 