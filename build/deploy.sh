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

set -o errexit -o nounset -o pipefail ${XTRACE:+-o xtrace}

until svcat version | grep -m 1 'Server Version: v' ; do
  sleep 1;
done

if ! kubectl get namespace minibroker 1> /dev/null 2> /dev/null; then
  kubectl create namespace minibroker
fi

helm upgrade minibroker \
  --install \
  --namespace minibroker \
  --wait \
  --set "image=${IMAGE}:${TAG}" \
  --set "imagePullPolicy=${IMAGE_PULL_POLICY}" \
  --set "deploymentStrategy=Recreate" \
  --set "logLevel=${LOG_LEVEL:-4}" \
  ${CLOUDFOUNDRY:+--set "deployServiceCatalog=false"} \
  ${CLOUDFOUNDRY:+--set "defaultNamespace=minibroker"} \
  "${OUTPUT_CHARTS_DIR}/minibroker-${TAG}.tgz"
