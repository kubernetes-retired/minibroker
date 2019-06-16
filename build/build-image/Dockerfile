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

FROM golang:1.10-stretch

ENV USE_DOCKER=false

WORKDIR /tmp

# Install tree
RUN apt-get update
RUN apt-get install -y tree

# Install Helm
ENV HELM_VERSION=2.13.1
RUN curl -sLO https://kubernetes-helm.storage.googleapis.com/helm-v${HELM_VERSION}-linux-amd64.tar.gz && \
    tar -zxvf helm-v${HELM_VERSION}-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/local/bin/

# Install kubectl
ENV KUBE_VERSION=1.9.6
RUN curl -sLO https://storage.googleapis.com/kubernetes-release/release/v${KUBE_VERSION}/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    mv ./kubectl /usr/local/bin/kubectl

# Install svcat
RUN curl -sLO https://download.svcat.sh/cli/latest/linux/amd64/svcat && \
    chmod +x ./svcat && \
    mv ./svcat /usr/local/bin/

# Install dep
ENV DEP_VERSION=v0.4.1
RUN curl -sLO https://github.com/golang/dep/releases/download/$DEP_VERSION/dep-linux-amd64 && \
    chmod +x ./dep-linux-amd64 && \
    mv ./dep-linux-amd64 /usr/local/bin/dep

# Install the azure client, used to publish svcat binaries
ENV AZCLI_VERSION=v0.3.1
RUN curl -sLo /usr/local/bin/az https://github.com/carolynvs/az-cli/releases/download/$AZCLI_VERSION/az-linux-amd64 && \
chmod +x /usr/local/bin/az

WORKDIR /go/src/github.com/osbkit/minibroker
