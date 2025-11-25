FROM golang:1.24.4-alpine AS builder

WORKDIR /build 

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download 

# Copy source code
COPY . .

# Build the simulator
RUN CGO_ENABLED=0 GOOS=linux go build -o simulator main.go

FROM alpine:latest

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/simulator .
COPY --from=builder /build/topology.yaml .

# Run the simulator
CMD ["./simulator"] 