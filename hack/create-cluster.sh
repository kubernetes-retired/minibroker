#!/usr/bin/env bash

# Copyright 2019 The Kubernetes Authors.
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

: "${VM_DRIVER:=docker}"
: "${VM_CPUS:=2}"
: "${VM_MEMORY:=$(( 1024 * 4 ))}"
: "${KUBERNETES_VERSION:=v1.15.12}"

export MINIKUBE_WANTUPDATENOTIFICATION=false

if [[ "$(minikube status)" != *"Running"* ]]; then
    set -o xtrace
    minikube start \
      --driver="${VM_DRIVER}" \
      --cpus="${VM_CPUS}" \
      --memory="${VM_MEMORY}" \
      --kubernetes-version="${KUBERNETES_VERSION}"
else
    >&2 echo "A Minikube instance is already running..."
    exit 1
fi

catalog_repository="svc-cat"
catalog_release="catalog"
catalog_namespace="svc-cat"
helm repo add "${catalog_repository}" https://kubernetes-sigs.github.io/service-catalog
helm repo update
kubectl create namespace "${catalog_namespace}"
helm install "${catalog_release}" \
  --namespace "${catalog_namespace}" \
  "${catalog_repository}/catalog"

set +o xtrace
while [[ "$(kubectl get pods --namespace "${catalog_namespace}" --selector "release=${catalog_release}" --output=go-template='{{.items | len}}')" == 0 ]]; do
  sleep 1;
done
set -o xtrace
kubectl wait pods \
  --for condition=ready \
  --namespace "${catalog_namespace}" \
  --selector "release=${catalog_release}" \
  --timeout 90s
