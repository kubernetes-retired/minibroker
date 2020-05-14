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

if [[ "$(minikube status)" != *"Running"* ]]; then
    set -o xtrace
    minikube start \
      --vm-driver="${VM_DRIVER}" \
      --cpus="${VM_CPUS}" \
      --memory="${VM_MEMORY}" \
      --kubernetes-version="${KUBERNETES_VERSION}"
else
    >&2 echo "A Minikube instance is already running..."
    exit 1
fi


kubectl apply -f https://raw.githubusercontent.com/Azure/helm-charts/master/docs/prerequisities/helm-rbac-config.yaml
helm init --service-account tiller --wait

helm repo add svc-cat https://svc-catalog-charts.storage.googleapis.com
helm upgrade --install catalog --namespace svc-cat svc-cat/catalog --wait
