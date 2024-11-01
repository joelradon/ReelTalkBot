# Use the official Golang image for building
FROM golang:1.22 AS builder
WORKDIR /app

# Copy go.mod and go.sum to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/main.go

# Use a minimal base image for the final stage
FROM debian:stable-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /root/
COPY --from=builder /app/main ./
RUN chmod +x ./main

# Expose the port
EXPOSE 8080

# Set environment variables (if needed)
# ENV AWS_ACCESS_KEY_ID=your_access_key_id
# ENV AWS_SECRET_ACCESS_KEY=your_secret_access_key
# ENV AWS_REGION=us-east-1
# ENV AWS_ENDPOINT_URL_S3=https://domain.dev
# ENV BUCKET_NAME=bucketname

# Run the Go app
CMD ["./main"]
