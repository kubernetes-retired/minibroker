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
REGISTRY ?= quay.io/kubernetes-service-catalog/
IMAGE ?= $(REGISTRY)minibroker
DATE ?= $(shell date --utc)
TAG ?= canary
IMAGE_PULL_POLICY ?= Never

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

build-image:
	docker build -t minibroker-build ./build/build-image

image:
	BUILD_IN_MINIKUBE=0 IMAGE="$(IMAGE)" TAG="$(TAG)" ./build/image.sh

image-in-minikube:
	BUILD_IN_MINIKUBE=1 IMAGE="$(IMAGE)" TAG="$(TAG)" ./build/image.sh

clean:
	-rm -rf $(OUTPUT_DIR)

push: image
	IMAGE=$(IMAGE) TAG=$(TAG) ./build/publish-images.sh

verify: verify-boilerplate

verify-boilerplate:
	./build/verify-boilerplate.sh

test-unit:
	ginkgo -cover cmd/... pkg/...

test-integration:
	(cd ./tests/integration; NAMESPACE=minibroker-tests ginkgo --nodes 4 --slowSpecThreshold 180 .)

test: test-unit test-integration test-wordpress

test-wordpress: setup-wordpress teardown-wordpress

setup-wordpress:
	kubectl create namespace minibroker-test-wordpress
	helm install minipress charts/wordpress --namespace minibroker-test-wordpress --wait

teardown-wordpress:
	helm delete minipress --namespace minibroker-test-wordpress
	kubectl delete namespace minibroker-test-wordpress

log:
	kubectl log -n minibroker deploy/minibroker-minibroker -c minibroker

create-cluster:
	./hack/create-cluster.sh

deploy: image-in-minikube
	IMAGE=$(IMAGE) TAG=$(TAG) IMAGE_PULL_POLICY=$(IMAGE_PULL_POLICY) ./build/deploy.sh

release: release-images release-charts

release-images: push

release-charts: build-image
	docker run --rm -it -v `pwd`:/go/src/$(REPO) -e AZURE_STORAGE_CONNECTION_STRING minibroker-build ./build/publish-charts.sh


.PHONY: build log build-linux test image clean push create-cluster deploy release
