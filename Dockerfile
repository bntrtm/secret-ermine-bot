# Use debian as the base image
FROM debian:stable-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*

# COPY source destination
COPY secret-ermine-bot /bin/seb

ENV PORT=8080

CMD ["/bin/seb"]
