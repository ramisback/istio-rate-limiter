# Build stage: Use golang:1.21-alpine as the base image for building the application
# Alpine-based image for smaller size and better security
FROM golang:1.21-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy all source code and configuration files into the container
COPY . .

# Download all required Go dependencies
RUN go mod download

# Build the application with CGO disabled for better compatibility
# and target Linux as the operating system
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage: Use alpine:latest as the base image for running the application
# This creates a minimal production image
FROM alpine:latest

# Set the working directory inside the container
WORKDIR /app

# Copy only the built binary from the builder stage
# This reduces the final image size significantly
COPY --from=builder /app/main .

# Expose port 8081 for the gRPC server
# Note: This port is used for the rate limit service's gRPC endpoint
EXPOSE 8081

# Command to run the application
CMD ["./main"] 