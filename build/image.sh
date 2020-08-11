#!/usr/bin/env bash

# Copyright 2020 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit -o nounset -o pipefail

if [[ "${BUILD_IN_MINIKUBE}" == "1" ]]; then
  if ! type -t minikube &>/dev/null; then
    >&2 echo "minikube not found in \$PATH"
    exit 1
  fi
  # shellcheck disable=SC2046
  eval $(minikube -p minikube docker-env)
fi

docker build \
  --tag "${IMAGE}:${TAG}" \
  ${BUILDER_IMAGE:+--build-arg "BUILDER_IMAGE=${BUILDER_IMAGE}"} \
  ${DOWNLOADER_IMAGE:+--build-arg "DOWNLOADER_IMAGE=${DOWNLOADER_IMAGE}"} \
  ${CERT_BUILDER_IMAGE:+--build-arg "CERT_BUILDER_IMAGE=${CERT_BUILDER_IMAGE}"} \
  ${RUNNING_IMAGE:+--build-arg "RUNNING_IMAGE=${RUNNING_IMAGE}"} \
  --build-arg "TAG=${TAG}" \
  --file docker/Dockerfile .
