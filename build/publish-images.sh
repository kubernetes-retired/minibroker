#!/usr/bin/env bash

set -euo pipefail

IMAGE=${IMAGE:-carolynvs/minibroker}
TAG=${TAG:-canary}

if [[ -v DOCKER_PASSWORD && -v DOCKER_USERNAME ]]; then
    echo "$DOCKER_PASSWORD" | docker login -u "$DOCKER_USERNAME" --password-stdin
fi

docker push "$IMAGE:$TAG"