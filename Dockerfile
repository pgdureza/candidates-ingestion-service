# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Dependencies
RUN apk add --no-cache git gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o api ./cmd/api && \
    CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o worker ./cmd/worker

# Runtime stage
FROM alpine:latest as runtime

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/api .
COPY --from=builder /app/worker .

EXPOSE 8080

# Default to API; override with: docker run [...] ./worker
CMD ["./api"]
