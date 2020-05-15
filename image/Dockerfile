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

# --------------------------------------------------------------------------------------------------
# The building stage.
FROM golang:1.14.2-stretch AS builder

WORKDIR /build/minibroker
# Copy the go.mod over so docker can cache the module downloads if possible.
COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./
RUN make build-linux

# --------------------------------------------------------------------------------------------------
# The downloading stage.
FROM alpine:latest AS downloader

ENV HELM_VERSION="v2.13.1"
ENV FILENAME="helm-${HELM_VERSION}-linux-amd64.tar.gz"

RUN wget -O /tmp/${FILENAME} http://storage.googleapis.com/kubernetes-helm/${FILENAME}
RUN tar -zxvf /tmp/${FILENAME} -C /tmp

# --------------------------------------------------------------------------------------------------
# The running stage.
FROM debian:stretch

RUN apt update && apt install -y ca-certificates && apt clean && rm -rf /var/lib/apt/lists/*

#RUN adduser -D minibroker
#USER minibroker

COPY --from=downloader /tmp/linux-amd64/helm /usr/local/bin/helm
RUN helm init --client-only

COPY --from=builder /build/minibroker/output/minibroker-linux /usr/local/bin/minibroker

CMD ["minibroker", "--help"]
