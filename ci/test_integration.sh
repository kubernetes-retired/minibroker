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

timeout 3m minikube start \
    --cpus="$(nproc)" \
    --memory=6g \
    --kubernetes-version=v1.21.8

catalog_repository="svc-cat"
catalog_release="catalog"
catalog_namespace="svc-cat"
helm repo add "${catalog_repository}" https://kubernetes-sigs.github.io/service-catalog
timeout 30s helm repo update
timeout 1m helm install "${catalog_release}" \
    --namespace "${catalog_namespace}" \
    --create-namespace \
    --wait \
    "${catalog_repository}/catalog"

timeout 1m kubectl create namespace minibroker-tests
timeout 10m make image
timeout 1m make charts
timeout 1m make minikube-load-image
timeout 5m make deploy
timeout 15m make test-integration
