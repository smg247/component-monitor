FROM golang:1.24-alpine

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN go build -o dashboard ./cmd/dashboard

# Expose the default port
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/app/dashboard"]

