#!/bin/bash
set -e

cd "$(dirname "$0")/.."

go build
docker-compose up -d --build --remove-orphans
