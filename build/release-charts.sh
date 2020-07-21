#!/usr/bin/env bash

# Copyright 2019-2020 The Kubernetes Authors.
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

CHART_REPOSITORY_ROOT=https://minibroker.blob.core.windows.net
AZURE_STORAGE_CONTAINER=charts
index_url="${CHART_REPOSITORY_ROOT}/${AZURE_STORAGE_CONTAINER}"

>&2 echo "Generating final index.yaml..."
helm repo index \
    --url "${index_url}" \
    --merge <(curl -L --silent "${index_url}/index.yaml") \
    "${OUTPUT_CHARTS_DIR}"

if [ ! -v AZURE_STORAGE_CONNECTION_STRING ]; then
    >&2 echo "AZURE_STORAGE_CONNECTION_STRING env var required to publish"
    exit 1
fi

>&2 echo "Uploading from ${OUTPUT_CHARTS_DIR}"
az storage blob upload-batch \
    --destination "${AZURE_STORAGE_CONTAINER}" \
    --source "${OUTPUT_CHARTS_DIR}"
