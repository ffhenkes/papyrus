# Stage 1: Build
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod ./
# Download dependencies (go.sum is generated automatically)
RUN go mod download || true

COPY main.go ./
RUN go mod tidy && go build -o papyrus -ldflags="-s -w" .

# Stage 2: Minimal runtime image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/papyrus .

VOLUME ["/pdfs"]

ENTRYPOINT ["./papyrus"]
