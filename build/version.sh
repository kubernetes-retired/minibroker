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

git_dirty=$([[ -z $(git status --short) ]] || echo "-dirty")
git_tag=$(git tag --points-at HEAD)
if [ -n "${git_tag}" ]; then
    # Use the git tag, removing the leading 'v' if it exists.
    echo "${git_tag/#v/}${git_dirty}"
else
    # No git tag found for current commit, the version is unknown, but we use
    # the short hash for reference.
    git_hash_short=$(git rev-parse --short HEAD)
    echo "0.0.0-${git_hash_short}${git_dirty}"
fi
