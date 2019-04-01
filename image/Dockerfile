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
