#!/usr/bin/env bash

set -o errexit -o nounset -o pipefail

if [[ "${BUILD_IN_MINIKUBE}" == "1" ]]; then
  # shellcheck disable=SC2046
  eval $(minikube -p minikube docker-env)
fi

docker build --tag "${IMAGE}:${TAG}" --file image/Dockerfile .
