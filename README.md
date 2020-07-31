[![Build Status](https://travis-ci.org/kubernetes-sigs/minibroker.svg?branch=master)](https://travis-ci.org/kubernetes-sigs/minibroker)

# Minibroker

> A minibroker for your minikube!

Minibroker is an implementation of the [Open Service Broker API](https://openservicebrokerapi.org)
suited for local development and testing. Rather than provisioning services
from a cloud provider, Minibroker provisions services in containers on the cluster.

Minibroker uses the [Kubernetes Helm Charts](https://github.com/kubernetes/charts)
as its source of provisionable services.

While it can deploy any stable chart, Minibroker provides the following Service Catalog Enabled
services:

* mysql
* postgres
* mariadb
* mongodb
* redis
* rabbitmq

Minibroker has built-in support for these charts so that the credentials are formatted
in a format that Service Catalog Ready charts expect.

# Prerequisites

* Kubernetes 1.9+ cluster
* [Helm 3](https://helm.sh)
* [Service Catalog](https://svc-cat.io/docs/install)
* [Service Catalog CLI (svcat)](http://svc-cat.io/docs/install/#installing-the-service-catalog-cli)

Run the following commands to set up a cluster:

```
minikube start

helm repo add svc-cat https://svc-catalog-charts.storage.googleapis.com
kubectl create namespace svc-cat
helm install catalog --namespace svc-cat svc-cat/catalog
```

# Install Minibroker

```
helm repo add minibroker https://minibroker.blob.core.windows.net/charts
kubectl create namespace minibroker
helm install minibroker --namespace minibroker minibroker/minibroker
```

*NOTE*: Platform users provisioning service instances will be able to set
arbitrary parameters, which can be potentially dangerous, e.g. if setting a
high number of replicas.
To prevent this, it is possible to define override parameters per service using
the `overrideChartParams` value. If defined, the user-defined parameters
are dropped and the override parameters are used instead.

## Installation Options
* Only Service Catalog Enabled services are included with Minibroker by default,
  to include all available charts specify `--set serviceCatalogEnabledOnly=false`.
* The stable Helm chart repository is the default source for services, to change
  the source Helm repository, specify
  `--set helmRepoUrl=https://example.com/custom-chart-repo/`.

# Update Minibroker

```
helm upgrade minibroker minibroker/minibroker \
  --install \
  --set deploymentStrategy="Recreate"
```

# Usage with Cloud Foundry

The Open Service Broker API is compatible with Cloud Foundry, and minibroker
can be used to respond to requests from a CF system.

## Installation

CF doesn't require the Service Catalog to be installed. The Cloud Controller,
which is part of the CFAR (Clouf Foundry Application Runtime), is the Platform
as specified in the OSBAPI.

```
helm repo add minibroker https://minibroker.blob.core.windows.net/charts
kubectl create namespace minibroker
helm install minibroker minibroker/minibroker \
  --namespace minibroker \
  --set "deployServiceCatalog=false" \
  --set "defaultNamespace=minibroker"
```

## Usage

The following usage instructions assume a successful login to the CF system,
with an Org and Space available. It also assumes a CF system like [KubeCF](https://github.com/cloudfoundry-incubator/kubecf)
that runs in the same Kubernetes cluster as the minibroker. It should be
possible to run the minibroker separately, but this would need a proper
ingress setup.

```
cf create-service-broker minibroker user pass http://minibroker-minibroker.minibroker.svc
cf enable-service-access redis
echo > redis.json '[{ "protocol": "tcp", "destination": "10.0.0.0/8", "ports": "6379", "description": "Allow Redis traffic" }]'
cf create-security-group redis_networking redis.json
cf bind-security-group   redis_networking org space
cf create-service redis 4-0-10 redis-example-svc
```

The service is then available for users of the CF system.

```
git clone https://github.com/scf-samples/cf-redis-example-app
cd cf-redis-example-app
cf push --no-start
cf bind-service redis-example-app redis-example-svc
cf start redis-example-app
```

The app can then be tested to confirm it can access the Redis service.

```
export APP=redis-example-app.cf-dev.io
curl -X GET $APP/foo # Returns 'key not present'
curl -X PUT $APP/foo -d 'data=bar'
curl -X GET $APP/foo # Returns 'bar'
```

# Examples

```
$ svcat get classes
     NAME             DESCRIPTION
+------------+---------------------------+
  mariadb      Helm Chart for mariadb
  mongodb      Helm Chart for mongodb
  mysql        Helm Chart for mysql
  postgresql   Helm Chart for postgresql

$ svcat describe class mysql
  Name:          mysql
  Description:   Helm Chart for mysql
  UUID:          mysql
  Status:        Active
  Tags:
  Broker:        minibroker

Plans:
   NAME             DESCRIPTION
+--------+--------------------------------+
  5-7-14   Fast, reliable, scalable,
           and easy to use open-source
           relational database system.

$ svcat provision mysqldb --class mysql --plan 5-7-14 -p mysqlDatabase=mydb -p mysqlUser=admin
  Name:        mysqldb
  Namespace:   minibroker
  Status:
  Class:       mysql
  Plan:        5-7-14

Parameters:
  mysqlDatabase: mydb
  mysqlUser: admin

$ svcat bind mysqldb
  Name:        mysqldb
  Namespace:   minibroker
  Status:
  Secret:      mysqldb
  Instance:    mysqldb

$ svcat describe binding mysqldb --show-secrets
  Name:        mysqldb
  Namespace:   minibroker
  Status:      Ready - Injected bind result @ 2018-04-27 03:53:09 +0000 UTC
  Secret:      mysqldb
  Instance:    mysqldb

Parameters:
  {}

Secret Data:
  database              mydb
  host                  lucky-dragon-mysql.minibroker.svc
  mysql-password        gsIpB8dBEn
  mysql-root-password   F8aBHuo8zb
  password              gsIpB8dBEn
  port                  3306
  uri                   mysql://admin:gsIpB8dBEn@lucky-dragon-mysql.minibroker.svc:3306/mydb
  username              admin

$ svcat unbind mysqldb
$ svcat deprovision mysqldb
```

To see Minibroker in action try out our Wordpress chart, that relies on Minibroker
to supply a database:

```
helm install minipress minibroker/wordpress
```

Follow the instructions output to the console to log into Wordpress.

## Helm Chart Parameters
Minibroker passes parameters specified during provisioning to the underlying
Helm Chart. This lets you customize the service to specify a non-root user, or the name of
the database to create, etc.

# Local Development

## Requirements

* Docker
* [Minikube](https://minikube.sigs.k8s.io/)
* [Helm 3](https://helm.sh)
* [Service Catalog CLI (svcat)](http://svc-cat.io/docs/install/#installing-the-service-catalog-cli)

## Setup

1. Create a Minikube cluster for local development by running `make create-cluster`. It defaults to
  using Docker as a VM driver. If you want to use a different VM driver, set the `VM_DRIVER`
  environment variable. E.g. `VM_DRIVER=kvm2 make create-cluster`.
2. Point your Docker to use the Minikube Docker daemon on the current shell session by running
  `eval $(minikube docker-env)`.

## Deploy

Compile and deploy the broker to your local cluster by running
`IMAGE_PULL_POLICY="Never" make image deploy`.

## Test

`make test`

There is an example chart for Wordpress that has been tweaked to use Minibroker for the
database provider, run `make setup-wordpress` to try it out.
