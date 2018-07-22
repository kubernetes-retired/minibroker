FROM golang:1.10-stretch

ENV USE_DOCKER=false

WORKDIR /tmp

# Install tree
RUN apt-get update
RUN apt-get install -y tree

# Install Helm
ENV HELM_VERSION=2.9.0
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
ENV AZCLI_VERSION=v0.1.2
RUN curl -sLo /usr/local/bin/az https://github.com/carolynvs/az-cli/releases/download/$AZCLI_VERSION/az-linux-amd64 && \
chmod +x /usr/local/bin/az

WORKDIR /go/src/github.com/osbkit/minibroker
