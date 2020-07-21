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

# Install helm.
version="v3.2.1"
sha256="018f9908cb950701a5d59e757653a790c66d8eda288625dbb185354ca6f41f6b"
asset_path="${HOME}/assets/helm.tar.gz"
asset_url="https://get.helm.sh/helm-${version}-linux-amd64.tar.gz"
if [ ! -f "${asset_path}" ] || [[ "$(sha256sum "${asset_path}" | awk '{ print $1 }')" != "${sha256}" ]]; then
  curl -Lo "${asset_path}" "${asset_url}"
fi
sudo tar zxf "${asset_path}" --strip-components=1 --directory /usr/local/bin/ linux-amd64/helm

# Install Azure CLI.
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
