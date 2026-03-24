# Stage 1: Build
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod ./
RUN go mod download || true

COPY main.go ./
RUN go mod tidy && go build -o papyrus -ldflags="-s -w" .

# Stage 2: Minimal runtime image
FROM alpine:3.19

# Install bash (required for script), curl (for Ollama health check), and dos2unix (for Windows line endings)
RUN apk add --no-cache ca-certificates bash curl dos2unix

WORKDIR /app

COPY --from=builder /app/papyrus .

COPY papyrus.sh .
# Ensure the script has Linux line endings and is executable
RUN dos2unix papyrus.sh && chmod +x papyrus.sh

VOLUME ["/pdfs"]

ENTRYPOINT ["/bin/bash", "./papyrus.sh"]
