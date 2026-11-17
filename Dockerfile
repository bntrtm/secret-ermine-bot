# Build binary
FROM --platform=$BUILDPLATFORM golang:1.27 AS builder

# buildx
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o secret-ermine-bot .

# Use debian as the base image
FROM debian:stable-slim

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Define startup command
WORKDIR /app
COPY --from=builder /app/secret-ermine-bot /bin/seb
EXPOSE 8080
ENTRYPOINT ["seb"]

