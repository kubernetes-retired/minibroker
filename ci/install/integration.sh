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

# Install kubectl.
version="v1.18.1"
sha256="f5144823e6d8a0b78611a8d12e7a25202126d079c3a232b18f37e61e872ff563"
asset_path="${HOME}/assets/kubectl"
asset_url="https://dl.k8s.io/release/${version}/bin/linux/amd64/kubectl"
if [ ! -f "${asset_path}" ] || [[ "$(sha256sum "${asset_path}" | awk '{ print $1 }')" != "${sha256}" ]]; then
  curl -Lo "${asset_path}" "${asset_url}"
  chmod +x "${asset_path}"
fi
sudo cp "${asset_path}" /usr/local/bin/kubectl

# Install minikube.
version="v1.10.1"
sha256="acc67ea2ff1ca261269a702a6d998367f65c86d9024c20bbf5ac3922bfca1aaa"
asset_path="${HOME}/assets/minikube"
asset_url="https://storage.googleapis.com/minikube/releases/${version}/minikube-linux-amd64"
if [ ! -f "${asset_path}" ] || [[ "$(sha256sum "${asset_path}" | awk '{ print $1 }')" != "${sha256}" ]]; then
  curl -Lo "${asset_path}" "${asset_url}"
  chmod +x "${asset_path}"
fi
sudo cp "${asset_path}" /usr/local/bin/minikube

# Install helm.
version="v3.2.1"
sha256="018f9908cb950701a5d59e757653a790c66d8eda288625dbb185354ca6f41f6b"
asset_path="${HOME}/assets/helm.tar.gz"
asset_url="https://get.helm.sh/helm-${version}-linux-amd64.tar.gz"
if [ ! -f "${asset_path}" ] || [[ "$(sha256sum "${asset_path}" | awk '{ print $1 }')" != "${sha256}" ]]; then
  curl -Lo "${asset_path}" "${asset_url}"
fi
sudo tar zxf "${asset_path}" --strip-components=1 --directory /usr/local/bin/ linux-amd64/helm

# Install svcat.
version="v0.3.0"
sha256="84ec798e8837982dfe13e5a02bf83e801af2461323ab2c441787d7d9f7bad60a"
asset_path="${HOME}/assets/svcat"
asset_url="https://download.svcat.sh/cli/${version}/linux/amd64/svcat"
if [ ! -f "${asset_path}" ] || [[ "$(sha256sum "${asset_path}" | awk '{ print $1 }')" != "${sha256}" ]]; then
  curl -Lo "${asset_path}" "${asset_url}"
  chmod +x "${asset_path}"
fi
sudo cp "${asset_path}" /usr/local/bin/svcat
