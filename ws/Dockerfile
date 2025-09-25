# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o wss .

# Final stage
FROM alpine:3.18

# Create restricted user "coder"
RUN addgroup -S coder && adduser -S coder -G coder -h /home/coder

# Create workspace dir
RUN mkdir -p /home/coder/workspace && chown -R coder:coder /home/coder

WORKDIR /home/coder/workspace

COPY --from=builder /app/wss /usr/local/bin/wss

USER coder

EXPOSE 8080
CMD ["wss"]

