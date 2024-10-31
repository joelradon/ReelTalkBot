# Use the official Golang image for building
FROM golang:1.22 AS builder
WORKDIR /app

# Copy go.mod and go.sum to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code and build
COPY . .
RUN go build -o main ./cmd

# Use a minimal base image for the final stage
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/main .

# Expose the port
EXPOSE 8080

# Run the Go app
CMD ["./main"]
