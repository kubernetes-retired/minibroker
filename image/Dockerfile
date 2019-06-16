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

FROM debian:stretch

RUN apt-get update
RUN apt-get install -y curl ca-certificates

ENV HELM_VERSION="v2.13.1"
ENV FILENAME="helm-${HELM_VERSION}-linux-amd64.tar.gz"

RUN curl -L http://storage.googleapis.com/kubernetes-helm/${FILENAME} -o /tmp/${FILENAME} && \
    tar -zxvf /tmp/${FILENAME} -C /tmp && \
    mv /tmp/linux-amd64/helm /usr/local/bin/

COPY minibroker /usr/local/bin/

#RUN adduser -D minibroker
#USER minibroker

RUN helm init --client-only

CMD ["minibroker", "--help"]
