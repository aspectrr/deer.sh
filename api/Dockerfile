FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy proto dependency first
COPY proto/gen/go ../proto/gen/go

# Copy api module
COPY api/go.mod api/go.sum ./
RUN go mod download
COPY api/ .
RUN CGO_ENABLED=0 go build -o /api ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /api /usr/local/bin/api

EXPOSE 8080 9090
ENTRYPOINT ["api"]
