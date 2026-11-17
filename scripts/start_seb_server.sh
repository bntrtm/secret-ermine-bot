#!/bin/bash
set -e

cd "$(dirname "$0")/.."

go build
docker build . -t secret-ermine-bot:latest
docker-compose down --remove-orphans
docker-compose up -d --build
