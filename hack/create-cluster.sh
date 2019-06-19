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

set -euo pipefail

: "${VM_DRIVER:=virtualbox}"
: "${VM_MEMORY:=$(( 1024 * 4 ))}"

if [[ "$(minikube status)" != *"Running"* ]]; then
    set -x
    minikube start \
      --vm-driver="${VM_DRIVER}" \
      --memory="${VM_MEMORY}" \
      --kubernetes-version=v1.11.3 \
      --bootstrapper=kubeadm
else
    echo "Using current running instance of Minikube..."
    set -x
fi

minikube addons enable heapster

kubectl apply -f https://raw.githubusercontent.com/Azure/helm-charts/master/docs/prerequisities/helm-rbac-config.yaml
helm init --service-account tiller --wait

helm repo add svc-cat https://svc-catalog-charts.storage.googleapis.com
helm upgrade --install catalog --namespace svc-cat svc-cat/catalog --wait
