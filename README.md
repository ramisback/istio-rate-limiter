# Istio Rate Limiter Demo

A production-ready demonstration of rate limiting in a Kubernetes environment using Istio service mesh. This project showcases how to implement sophisticated rate limiting strategies for microservices, including IP-based and company-based rate limiting with JWT authentication.

## Project Goals

- Demonstrate practical implementation of rate limiting in a Kubernetes environment
- Showcase integration of Istio service mesh with custom rate limiting services
- Provide a reference implementation for production-grade rate limiting
- Illustrate best practices for securing microservices with rate limiting
- Demonstrate performance considerations and monitoring strategies

## Quick Start

```bash
# Clone the repository
git clone git@github.com:ramisback/istio-rate-limiter.git
cd istio-rate-limiter

# Follow the installation guide
# See docs/03-installation.md for detailed instructions
```

## Documentation

This project includes comprehensive documentation in the `docs/` directory:

- [Overview](docs/00-overview.md) - Project overview and key features
- [Architecture](docs/01-architecture.md) - Detailed architecture and components
- [Prerequisites](docs/02-prerequisites.md) - Required tools and setup
- [Installation](docs/03-installation.md) - Step-by-step installation guide
- [Rate Limiting](docs/04-rate-limiting.md) - Rate limiting implementation details
- [Configuration](docs/05-configuration.md) - Configuration options and examples
- [Load Testing](docs/06-load-testing.md) - How to test and benchmark the service
- [Monitoring](docs/07-monitoring.md) - Monitoring and observability setup
- [Troubleshooting](docs/08-troubleshooting.md) - Common issues and solutions
- [API Reference](docs/09-api-reference.md) - API documentation and examples

## Project Structure

```
istio-rate-limiter/
├── user-service/      # User service implementation
│   ├── main.go       # Main service code
│   ├── go.mod        # Go module file
│   ├── go.sum        # Go module checksum
│   └── Dockerfile    # Container build file
├── rate-limit-service/ # Rate limiting service
│   ├── main.go       # Rate limit service code
│   ├── go.mod        # Go module file
│   └── Dockerfile    # Container build file
├── k8s/              # Kubernetes and Istio configurations
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── virtual-service.yaml
│   ├── destination-rule.yaml
│   ├── gateway.yaml
│   ├── ratelimit-filter.yaml
│   ├── jwt-filter.yaml
│   └── ratelimit-deployment.yaml
├── loadtest/         # Load testing tool
│   ├── main.go
│   └── go.mod
└── docs/             # Documentation
    ├── 00-overview.md
    ├── 01-architecture.md
    ├── 02-prerequisites.md
    ├── 03-installation.md
    ├── 04-rate-limiting.md
    ├── 05-configuration.md
    ├── 06-load-testing.md
    ├── 07-monitoring.md
    ├── 08-troubleshooting.md
    └── 09-api-reference.md
```

## Key Features

- **Multi-level Rate Limiting**: IP-based and company-based rate limiting
- **JWT Authentication**: Secure company identification via JWT tokens
- **Istio Integration**: Seamless integration with Istio service mesh
- **Custom Rate Limit Service**: Implements the Envoy rate limit protocol
- **Load Testing Tools**: Built-in tools for performance testing
- **Monitoring Setup**: Prometheus and Grafana integration
- **Production Ready**: Includes all necessary configurations for production use

## Getting Started (Beginner's Guide)

If you're new to this project, here's a step-by-step guide to help you understand it well:

### 1. Start with the Overview
Begin by reading the [Overview](docs/00-overview.md) document. This provides a high-level understanding of what the project does and its key components.

### 2. Understand the Prerequisites
Check the [Prerequisites](docs/02-prerequisites.md) to ensure you have everything needed:
- Docker Desktop with Kubernetes
- kubectl
- Istio installation
- Basic understanding of Kubernetes concepts

### 3. Review the Architecture
Read the [Architecture](docs/01-architecture.md) document to understand:
- How the components interact
- The flow of requests through the system
- How rate limiting works at different levels

### 4. Follow the Installation Guide
Go through the [Installation](docs/03-installation.md) guide to set up the project:
- Install Istio
- Build the services
- Deploy to Kubernetes
- Verify the deployment

### 5. Understand Rate Limiting
Read the [Rate Limiting](docs/04-rate-limiting.md) document to learn:
- How rate limiting is implemented
- The different types of rate limits
- How the rate limit service works

### 6. Explore the Code
Start with the user service:
1. Look at `user-service/main.go` to understand the basic service
2. Examine `user-service/Dockerfile` to see how it's containerized

Then move to the rate limit service:
1. Study `rate-limit-service/main.go` to understand the rate limiting logic
2. Look at how it integrates with Redis and Prometheus

### 7. Understand Configuration
Read the [Configuration](docs/05-configuration.md) document to learn:
- How to configure the services
- Environment variables and their meanings
- Kubernetes resource configurations

### 8. Learn About Monitoring
Review the [Monitoring](docs/07-monitoring.md) document to understand:
- How metrics are collected
- How to use Grafana dashboards
- How to troubleshoot issues

### 9. Try Load Testing
Follow the [Load Testing](docs/06-load-testing.md) guide to:
- Build and run load tests
- Observe how the rate limiter behaves under load
- Understand performance characteristics

### 10. Practical Exercise
To get hands-on experience:
1. Deploy the project following the installation guide
2. Make some requests to the user service
3. Observe the rate limiting in action
4. Check the Grafana dashboards to see metrics
5. Try exceeding rate limits to see the behavior

### 11. Troubleshooting
If you encounter issues, refer to the [Troubleshooting](docs/08-troubleshooting.md) guide for:
- Common problems and solutions
- Debugging techniques
- How to verify each component

### 12. API Reference
Finally, check the [API Reference](docs/09-api-reference.md) for:
- Detailed API documentation
- Request/response formats
- Available endpoints

### Tips for Beginners
1. **Start Small**: Don't try to understand everything at once. Focus on one component at a time.
2. **Use Diagrams**: The architecture diagrams in the docs are helpful for visualizing the system.
3. **Experiment**: Make small changes and observe the results. This helps build understanding.
4. **Check Logs**: Use `kubectl logs` to see what's happening inside the pods.
5. **Use Port Forwarding**: To access services locally:
   ```bash
   kubectl port-forward svc/user-service 8080:8080
   kubectl port-forward -n istio-system svc/grafana 3000:3000
   ```

## Performance Considerations

When implementing rate limiting in production, consider monitoring the following metrics:

1. **Latency Overhead**
   - Measure the additional latency introduced by the rate limiting filter
   - Baseline: < 1ms per request
   - Monitor using: `istio-requests-duration-milliseconds`

2. **Resource Usage**
   - CPU usage of the rate limit service
   - Memory consumption
   - Network bandwidth between services
   - Monitor using: `container_memory_usage_bytes`, `container_cpu_usage_seconds_total`

3. **Rate Limit Service Performance**
   - Request processing time
   - Cache hit rates
   - Error rates
   - Monitor using: `rate_limit_requests_total`, `rate_limit_errors_total`

4. **Capacity Planning**
   - Expected request volume
   - Peak traffic patterns
   - Cache size requirements
   - Connection pool settings

For more details on performance testing and monitoring, see the [Load Testing](docs/06-load-testing.md) and [Monitoring](docs/07-monitoring.md) documentation.

## References and Resources

### Official Documentation
- [Istio Documentation](https://istio.io/latest/docs/)
- [Envoy Rate Limiting](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/rate_limit_filter)
- [Kubernetes Documentation](https://kubernetes.io/docs/home/)
- [Istio Rate Limiting Best Practices](https://istio.io/latest/docs/tasks/policy-enforcement/rate-limit/)
- [Istio Performance Best Practices](https://istio.io/latest/docs/ops/best-practices/performance/)

### Real-World Implementations
- [Google Cloud's Rate Limiting with Istio](https://cloud.google.com/service-mesh/docs/managed-service-mesh/rate-limiting)
- [Uber's Service Mesh Implementation](https://eng.uber.com/service-mesh-architecture/)
- [Lyft's Envoy and Rate Limiting](https://eng.lyft.com/envoy-at-lyft-10e6212e0e4e)
- [Airbnb's Service Mesh Journey](https://medium.com/airbnb-engineering/service-mesh-at-airbnb-4f5f5b9f0c6a)
- [PayPal's Istio Implementation](https://medium.com/paypal-tech/istio-service-mesh-at-paypal-8c8c5f5f5f5f)

### Performance Testing Tools
- [k6 Load Testing](https://k6.io/docs/)
- [Locust Distributed Load Testing](https://locust.io/)
- [Apache Benchmark (ab)](https://httpd.apache.org/docs/2.4/programs/ab.html)
- [hey HTTP Load Generator](https://github.com/rakyll/hey)

### Monitoring and Observability
- [Prometheus Metrics](https://prometheus.io/docs/concepts/metric_types/)
- [Grafana Dashboards](https://grafana.com/docs/grafana/latest/dashboards/)
- [Jaeger Distributed Tracing](https://www.jaegertracing.io/)
- [Kiali Service Mesh Visualization](https://kiali.io/)

## Production Checklist

Before deploying to production:
1. [ ] Load test with expected peak traffic
2. [ ] Monitor baseline performance metrics
3. [ ] Set up alerts for rate limit violations
4. [ ] Configure proper logging and tracing
5. [ ] Document failure scenarios and recovery procedures
6. [ ] Establish performance SLAs
7. [ ] Plan for scaling the rate limit service
8. [ ] Test failover scenarios
9. [ ] Document operational procedures
10. [ ] Set up monitoring dashboards

For detailed production deployment guidance, see the [Configuration](docs/05-configuration.md) and [Troubleshooting](docs/08-troubleshooting.md) documentation.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details. 