#!/usr/bin/env bash

set -euo pipefail

SERVER=${SERVER:-quay.io}
IMAGE=${IMAGE:-quay.io/kubernetes-service-catalog/minibroker}
TAG=${TAG:-canary}

if [[ -v DOCKER_PASSWORD && -v DOCKER_USERNAME ]]; then
    echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin $SERVER
fi

docker push "$IMAGE:$TAG"
