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

ARG BUILDER_IMAGE=golang:1.14.2-buster@sha256:6e35cbd04ca339a53194a2542d07dfba0f1ec0185a6f69dc33310540cc38144f
ARG DOWNLOADER_IMAGE=alpine:latest
ARG CERT_BUILDER_IMAGE=opensuse/leap:15.1@sha256:ace17bbeacb203c4bf1d3d23ce7c9e2639fe0527a87e7c5758387e749943a49a
ARG RUNNING_IMAGE=debian:stable@sha256:e0635a846513de0357689a7dd0c605a538d4ba2093a14b4688a15ed509c88e55

# --------------------------------------------------------------------------------------------------
# The building stage.
FROM ${BUILDER_IMAGE} AS builder

WORKDIR /build/minibroker
# Copy the go.mod over so docker can cache the module downloads if possible.
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
COPY pkg/ ./pkg/
COPY Makefile ./
ARG TAG
ENV TAG $TAG
RUN make build

# --------------------------------------------------------------------------------------------------
# The downloading stage.
FROM ${DOWNLOADER_IMAGE} AS downloader

RUN wget -O /tmp/dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.2.2/dumb-init_1.2.2_amd64
RUN chmod +x /tmp/dumb-init

# --------------------------------------------------------------------------------------------------
# The cert building stage.
FROM ${CERT_BUILDER_IMAGE} AS cert_builder

ARG CURL_VERSION="7.70.0"

RUN zypper refresh
RUN zypper --non-interactive install perl-Encode make tar gzip curl

RUN curl -L -o /tmp/curl-${CURL_VERSION}.tar.gz https://github.com/curl/curl/releases/download/curl-${CURL_VERSION//\./_}/curl-${CURL_VERSION}.tar.gz
RUN tar zxf /tmp/curl-${CURL_VERSION}.tar.gz
WORKDIR /curl-${CURL_VERSION}

RUN make ca-bundle
RUN cp lib/ca-bundle.crt /tmp/ca-bundle.crt

# --------------------------------------------------------------------------------------------------
# The running stage.
FROM ${RUNNING_IMAGE}

COPY --from=cert_builder /tmp/ca-bundle.crt /etc/ssl/certs/ca-bundle.crt
COPY --from=downloader /tmp/dumb-init /usr/local/bin/dumb-init
COPY --from=builder /build/minibroker/output/minibroker /usr/local/bin/minibroker

COPY docker/rootfs/etc/passwd /etc/passwd
COPY --chown=1000 docker/rootfs/home/minibroker /home/minibroker
USER 1000

VOLUME /home/minibroker
ENV TMPDIR /home/minibroker/tmp
ENTRYPOINT ["dumb-init", "--"]
CMD ["minibroker", "--help"]
