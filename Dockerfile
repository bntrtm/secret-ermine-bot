# Build binary
FROM golang:1.27 AS builder
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o secret-ermine-bot .

# Use debian as the base image
FROM debian:stable-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*

# Define startup command
WORKDIR /app
COPY --from=builder /app/secret-ermine-bot /bin/seb
EXPOSE 8080
ENTRYPOINT ["seb"]

