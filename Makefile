REPO ?= github.com/osbkit/minibroker
BINARY ?= minibroker
PKG ?= $(REPO)/cmd/$(BINARY)
IMAGE ?= carolynvs/minibroker
TAG ?= latest

build:
	go build $(PKG)

test:
	go test -v ./...
	svcat get plans | grep db
	svcat provision mydb --class mariadb --plan 10-1-31 --namespace minibroker
	svcat get instances -n minibroker
	svcat bind mydb
	svcat get bindings -n minibroker


build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
	go build -o $(BINARY)-linux -tags netgo --ldflags="-s" $(PKG)

image: build-linux
	cp $(BINARY)-linux image/$(BINARY)
	docker build image/ -t "$(IMAGE):$(TAG)"

clean:
	-rm -f $(BINARY)

push: image
	docker push "$(IMAGE):$(TAG)"

create-cluster:
	./hack/create-cluster.sh

deploy: push
	helm upgrade --install minibroker --namespace minibroker \
	--recreate-pods --force charts/minibroker \
	--set image="$(IMAGE):$(TAG)",imagePullPolicy="Always",deploymentStrategy="Recreate"

.PHONY: build build-linux test image clean push create-cluster deploy
