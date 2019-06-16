#!/usr/bin/env bash

set -euo pipefail

# REPO_ROOT is used by verify-boilerplate.sh
export REPO_ROOT="$( cd "$( dirname "${BASH_SOURCE[0]}" )/.." && pwd )"

$REPO_ROOT/vendor/github.com/kubernetes/repo-infra/verify/verify-boilerplate.sh