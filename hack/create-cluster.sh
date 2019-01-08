#!/usr/bin/env bash

set -xeuo pipefail

minikube addons enable heapster

if [[ "$(minikube status)" != *"Running"* ]]; then
    minikube start --vm-driver=virtualbox \
    --kubernetes-version=v1.12.0 \
    --bootstrapper=kubeadm
fi

kubectl apply -f https://raw.githubusercontent.com/Azure/helm-charts/master/docs/prerequisities/helm-rbac-config.yaml
helm init --service-account tiller --wait

helm repo add svc-cat https://svc-catalog-charts.storage.googleapis.com
helm repo up
helm upgrade --install catalog --namespace svc-cat svc-cat/catalog --wait
