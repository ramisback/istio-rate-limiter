#!/bin/bash

# Get the external IP of the Istio ingress gateway
echo "Getting the external IP of the Istio ingress gateway..."
EXTERNAL_IP=$(kubectl get svc -n istio-system istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

if [ -z "$EXTERNAL_IP" ]; then
  echo "Could not get external IP. Trying alternative method..."
  EXTERNAL_IP=$(kubectl get svc -n istio-system istio-ingressgateway -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
  
  if [ -z "$EXTERNAL_IP" ]; then
    echo "Could not get external IP. Setting up port forwarding..."
    # Set up port forwarding in the background
    kubectl port-forward -n istio-system svc/istio-ingressgateway 8080:80 &
    PF_PID=$!
    echo "Port forwarding started with PID: $PF_PID"
    EXTERNAL_IP="http://localhost:8080"
    # Wait for port forwarding to be ready
    sleep 5
  else
    EXTERNAL_IP="http://$EXTERNAL_IP"
  fi
else
  EXTERNAL_IP="http://$EXTERNAL_IP"
fi

echo "Using external IP: $EXTERNAL_IP"

# Build the load test
echo "Building load test..."
cd loadtest
go build -o loadtest

# Run the load test
echo "Running load test..."
./loadtest -url "$EXTERNAL_IP" -rps 100 -duration 5m -concurrency 10

# Clean up port forwarding if it was started
if [ ! -z "$PF_PID" ]; then
  echo "Stopping port forwarding..."
  kill $PF_PID
fi 