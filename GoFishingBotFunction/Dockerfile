# Start from the official Go image with version 1.23
FROM golang:1.23-alpine

# Set the working directory
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the application
RUN go build -o main .

# Expose port 8080 and run the application
EXPOSE 8080
CMD ["./main"]
