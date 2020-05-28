#!/usr/bin/env bash

set -o errexit -o nounset -o pipefail

VM_DRIVER=none sudo timeout 3m make create-cluster
sudo chown -R "$(whoami):" "${HOME}/.minikube/"
timeout 5m make deploy
timeout 5m make test-integration
