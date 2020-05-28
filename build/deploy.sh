#!/usr/bin/env bash

set -o errexit -o nounset -o pipefail

until svcat version | grep -m 1 'Server Version: v' ; do
  sleep 1;
done

if ! kubectl get namespace minibroker 1> /dev/null 2> /dev/null; then
  kubectl create namespace minibroker
fi

helm upgrade minibroker \
  --install \
  --force \
  --recreate-pods \
  --namespace minibroker \
  --set "image=${IMAGE}:${TAG}" \
  --set "imagePullPolicy=${IMAGE_PULL_POLICY}" \
  --set "deploymentStrategy=Recreate" \
  charts/minibroker
