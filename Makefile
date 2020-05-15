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
TAG ?= canary
IMAGE_PULL_POLICY ?= Always

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(OUTPUT_DIR)/minibroker $(PKG)

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		go build -ldflags="-s -w" -o $(OUTPUT_DIR)/$(BINARY)-linux -tags netgo $(PKG)

build-image:
	docker build -t minibroker-build ./build/build-image

verify: verify-boilerplate

verify-boilerplate:
	./build/verify-boilerplate.sh

test-unit:
	go test -v ./cmd/... ./pkg/...

test: test-unit test-mariadb test-mysqldb test-postgresql test-mongodb test-wordpress

test-wordpress: setup-wordpress teardown-wordpress

setup-wordpress:
	helm install --name minipress charts/wordpress --wait

teardown-wordpress:
	helm delete --purge minipress

test-mysqldb: setup-mysqldb teardown-mysqldb

setup-mysqldb:
	until svcat get broker minibroker | grep -w -m 1 Ready; do : ; done

	svcat provision mysqldb --class mysql --plan 5-7-14 --namespace minibroker \
		-p mysqlDatabase=mydb -p mysqlUser=admin
	until svcat get instance mysqldb -n minibroker | grep -w -m 1 Ready; do : ; done
	svcat get instance mysqldb -n minibroker

	svcat bind mysqldb -n minibroker
	until svcat get binding mysqldb -n minibroker | grep -w -m 1 Ready; do : ; done
	svcat describe binding mysqldb -n minibroker

teardown-mysqldb:
	svcat unbind mysqldb -n minibroker
	svcat deprovision mysqldb -n minibroker

test-mariadb: setup-mariadb teardown-mariadb

setup-mariadb:
	until svcat get broker minibroker | grep -w -m 1 Ready; do : ; done

	svcat provision mariadb --class mariadb --plan 10-1-32 --namespace minibroker \
		--params-json '{"db": {"name": "mydb", "user": "admin"}}'
	until svcat get instance mariadb -n minibroker | grep -w -m 1 Ready; do : ; done
	svcat get instance mariadb -n minibroker

	svcat bind mariadb -n minibroker
	until svcat get binding mariadb -n minibroker | grep -w -m 1 Ready; do : ; done
	svcat describe binding mariadb -n minibroker

teardown-mariadb:
	svcat unbind mariadb -n minibroker
	svcat deprovision mariadb -n minibroker

test-postgresql: setup-postgresql teardown-postgresql

setup-postgresql:
	until svcat get broker minibroker | grep -w -m 1 Ready; do : ; done

	svcat provision postgresql --class postgresql --plan 11-0-0 --namespace minibroker \
		-p postgresDatabase=mydb -p postgresUser=admin
	until svcat get instance postgresql -n minibroker | grep -w -m 1 Ready; do : ; done
	svcat get instance postgresql -n minibroker

	svcat bind postgresql -n minibroker
	until svcat get binding postgresql -n minibroker | grep -w -m 1 Ready; do : ; done
	svcat describe binding postgresql -n minibroker

teardown-postgresql:
	svcat unbind postgresql -n minibroker
	svcat deprovision postgresql -n minibroker

test-mongodb: setup-mongodb teardown-mongodb

setup-mongodb:
	until svcat get broker minibroker | grep -w -m 1 Ready; do : ; done

	svcat provision mongodb --class mongodb --plan 3-7-1 --namespace minibroker \
		-p mongodbDatabase=mydb -p postgresUsername=admin
	until svcat get instance mongodb -n minibroker | grep -w -m 1 Ready; do : ; done
	svcat get instance mongodb -n minibroker

	svcat bind mongodb -n minibroker
	until svcat get binding mongodb -n minibroker | grep -w -m 1 Ready; do : ; done
	svcat describe binding mongodb -n minibroker

teardown-mongodb:
	svcat unbind mongodb -n minibroker
	svcat deprovision mongodb -n minibroker

image: build-linux
	cp $(BINARY)-linux image/$(BINARY)
	docker build image/ -t "$(IMAGE):$(TAG)"

clean:
	-rm -f $(BINARY)

push: image
	IMAGE=$(IMAGE) TAG=$(TAG) ./build/publish-images.sh

log:
	kubectl log -n minibroker deploy/minibroker-minibroker -c minibroker

create-cluster:
	./hack/create-cluster.sh

deploy:
	until svcat version | grep -m 1 'Server Version: v' ; do : ; done
	helm upgrade --install minibroker --namespace minibroker \
	--recreate-pods --force charts/minibroker \
	--set image="$(IMAGE):$(TAG)",imagePullPolicy="$(IMAGE_PULL_POLICY)",deploymentStrategy="Recreate"

release: release-images release-charts

release-images: push

release-charts: build-image
	docker run --rm -it -v `pwd`:/go/src/$(REPO) -e AZURE_STORAGE_CONNECTION_STRING minibroker-build ./build/publish-charts.sh


.PHONY: build log build-linux test image clean push create-cluster deploy release
