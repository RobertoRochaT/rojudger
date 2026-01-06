#!/bin/bash

set -e

echo "=========================================="
echo "  ROJUDGER - Docker Image Puller"
echo "=========================================="
echo ""

# List of required images
IMAGES=(
    "python:3.11-slim"
    "node:20-slim"
    "golang:1.21-alpine"
    "gcc:11"
)

echo "This script will pull the following Docker images:"
for image in "${IMAGES[@]}"; do
    echo "  - $image"
done
echo ""

# Pull each image
for image in "${IMAGES[@]}"; do
    echo "Pulling $image..."
    if docker pull "$image"; then
        echo "✓ Successfully pulled $image"
    else
        echo "✗ Failed to pull $image"
        exit 1
    fi
    echo ""
done

echo "=========================================="
echo "  All images pulled successfully!"
echo "=========================================="
echo ""
echo "Verifying images:"
docker images | grep -E "python|node|golang|gcc" || true
echo ""
echo "You can now run ROJUDGER without image pull delays."
