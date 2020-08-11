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

REPO ?= github.com/kubernetes-sigs/minibroker
BINARY ?= minibroker
PKG ?= $(REPO)/cmd/$(BINARY)
OUTPUT_DIR ?= output
OUTPUT_CHARTS_DIR ?= $(OUTPUT_DIR)/charts
REGISTRY ?= quay.io/kubernetes-service-catalog/
IMAGE ?= $(REGISTRY)minibroker
TAG ?= $(shell ./build/version.sh)
DATE ?= $(shell date --utc)
CHART_SIGN_KEY ?=
IMAGE_PULL_POLICY ?= Never
TMP_BUILD_DIR ?= tmp
WORDPRESS_CHART ?= $(shell pwd)/charts/wordpress

# The base images for the Dockerfile stages.
BUILDER_IMAGE ?= golang:1.14.2-buster@sha256:6e35cbd04ca339a53194a2542d07dfba0f1ec0185a6f69dc33310540cc38144f
DOWNLOADER_IMAGE ?= alpine:latest
CERT_BUILDER_IMAGE ?= opensuse/leap:15.1@sha256:ace17bbeacb203c4bf1d3d23ce7c9e2639fe0527a87e7c5758387e749943a49a
RUNNING_IMAGE ?= debian:stable@sha256:e0635a846513de0357689a7dd0c605a538d4ba2093a14b4688a15ed509c88e55

lint: lint-go-vet lint-go-mod lint-modified-files

lint-go-vet:
	go vet ./...

lint-go-mod:
	go mod tidy

lint-modified-files: | lint-go-mod generate
	./build/verify-modified-files.sh

generate:
	find . -type d -name '*mocks' -print -prune -exec rm -rf '{}' \;
	go generate ./...

build:
	CGO_ENABLED=0 go build -ldflags="-s -w -X 'main.version=$(TAG)' -X 'main.buildDate=$(DATE)'" -o $(OUTPUT_DIR)/minibroker $(PKG)

image:
	BUILD_IN_MINIKUBE=0 \
	BUILDER_IMAGE="$(BUILDER_IMAGE)" \
	DOWNLOADER_IMAGE="$(DOWNLOADER_IMAGE)" \
	CERT_BUILDER_IMAGE="$(CERT_BUILDER_IMAGE)" \
	RUNNING_IMAGE="$(RUNNING_IMAGE)" \
	IMAGE="$(IMAGE)" \
	TAG="$(TAG)" \
	./build/image.sh

image-in-minikube:
	BUILD_IN_MINIKUBE=1 \
	BUILDER_IMAGE="$(BUILDER_IMAGE)" \
	DOWNLOADER_IMAGE="$(DOWNLOADER_IMAGE)" \
	CERT_BUILDER_IMAGE="$(CERT_BUILDER_IMAGE)" \
	RUNNING_IMAGE="$(RUNNING_IMAGE)" \
	IMAGE="$(IMAGE)" \
	TAG="$(TAG)" \
	./build/image.sh

charts:
	CHART_SRC="charts/minibroker" \
	TMP_BUILD_DIR="$(TMP_BUILD_DIR)" \
	OUTPUT_CHARTS_DIR="$(OUTPUT_CHARTS_DIR)" \
	APP_VERSION="$(TAG)" \
	VERSION="$(TAG)" \
	CHART_SIGN_KEY="$(CHART_SIGN_KEY)" \
	IMAGE="$(IMAGE)" \
	TAG="$(TAG)" \
	./build/charts.sh

clean:
	-rm -rf "$(OUTPUT_DIR)"
	-rm -rf "$(TMP_BUILD_DIR)"

verify: verify-boilerplate

verify-boilerplate:
	./build/verify-boilerplate.sh

test-unit:
	ginkgo -cover cmd/... pkg/...

test-integration:
	(cd ./tests/integration; NAMESPACE=minibroker-tests WORDPRESS_CHART="$(WORDPRESS_CHART)" ginkgo --nodes 4 --slowSpecThreshold 180 .)

test: test-unit test-integration test-wordpress

log:
	kubectl log -n minibroker deploy/minibroker-minibroker -c minibroker

create-cluster:
	./hack/create-cluster.sh

deploy:
	IMAGE="$(IMAGE)" \
	TAG="$(TAG)" \
	IMAGE_PULL_POLICY="$(IMAGE_PULL_POLICY)" \
	OUTPUT_CHARTS_DIR="$(OUTPUT_CHARTS_DIR)" \
	./build/deploy.sh

deploy-cf:
	IMAGE="$(IMAGE)" \
	TAG="$(TAG)" \
	IMAGE_PULL_POLICY="$(IMAGE_PULL_POLICY)" \
	OUTPUT_CHARTS_DIR="$(OUTPUT_CHARTS_DIR)" \
	CLOUDFOUNDRY=true \
	./build/deploy.sh

deploy-dev: image-in-minikube charts deploy

deploy-dev-cf: image-in-minikube charts deploy-cf

release: clean release-images release-charts

release-images: image
	IMAGE="$(IMAGE)" TAG="$(TAG)" ./build/release-images.sh

release-charts: charts
	OUTPUT_CHARTS_DIR="$(OUTPUT_CHARTS_DIR)" ./build/release-charts.sh

.PHONY: build log test image charts clean push create-cluster deploy release
