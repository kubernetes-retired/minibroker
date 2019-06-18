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

Minibroker has built-in support for these charts so that the credentials are formatted
in a format that Service Catalog Ready charts expect.

# Prerequisites

* Kubernetes 1.9+ cluster
* [Helm](https://helm.sh)
* [Service Catalog](https://svc-cat.io/docs/install)
* [Service Catalog CLI (svcat)](http://svc-cat.io/docs/install/#installing-the-service-catalog-cli)

Run the following commands to set up a cluster:

```
minikube start --kubernetes-version=v1.9.6 --bootstrapper=kubeadm

kubectl apply -f https://raw.githubusercontent.com/Azure/helm-charts/master/docs/prerequisities/helm-rbac-config.yaml
helm init --service-account tiller --wait

helm repo add svc-cat https://svc-catalog-charts.storage.googleapis.com
helm install --name catalog --namespace svc-cat svc-cat/catalog --wait
```

# Install Minibroker

```
helm repo add minibroker https://minibroker.blob.core.windows.net/charts
helm install --name minibroker --namespace minibroker minibroker/minibroker
```

## Installation Options
* Only Service Catalog Enabled services are included with Minibroker by default,
  to include all available charts specify `--set serviceCatalogEnabledOnly=false`.
* The stable Helm chart repository is the default source for services, to change
  the source Helm repository, specify
  `--set helmRepoUrl=https://example.com/custom-chart-repo/`.

# Update Minibroker

```
helm upgrade --install minibroker \
	--recreate-pods --force minibroker/minibroker \
	--set imagePullPolicy="Always",deploymentStrategy="Recreate"
```

# Usage with Cloud Foundry

The Open Service Broker API is compatible with Cloud Foundry, and minibroker
can be used to respond to requests from a CF system.

## Installation

CF doesn't use a service catalog as the Cloud Controller handles the request
for services.

```
helm repo add minibroker https://minibroker.blob.core.windows.net/charts
helm install --name minibroker --namespace minibroker minibroker/minibroker \
	--set "deployServiceCatalog=false" \
        --set "defaultNamespace=minibroker"
```

## Usage

The following usage instructions assume a successful login to the CF system,
with an Org and Space available. It also assumes a CF system like [SUSE CAP](https://github.com/SUSE/scf)
that runs in the same Kubernetes cluster as the minibroker. It should be
possible to run the minibroker separately, but this would need a proper
ingress setup.

```
cf create-service-broker minibroker user pass http://minibroker-minibroker.minibroker.svc.cluster.local
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
  host                  lucky-dragon-mysql.minibroker.svc.cluster.local
  mysql-password        gsIpB8dBEn
  mysql-root-password   F8aBHuo8zb
  password              gsIpB8dBEn
  port                  3306
  uri                   mysql://admin:gsIpB8dBEn@lucky-dragon-mysql.minibroker.svc.cluster.local:3306/mydb
  username              admin

$ svcat unbind mysqldb
$ svcat deprovision mysqldb
```

To see Minibroker in action try out our Wordpress chart, that relies on Minibroker
to supply a database:

```
helm install --name minipress minibroker/wordpress
```

Follow the instructions output to the console to log into Wordpress.

## Helm Chart Parameters
Minibroker passes parameters specified during provisioning to the underlying
Helm Chart. This lets you customize the service to specify a non-root user, or the name of
the database to create, etc.

# Local Development

## Requirements

* Docker
* [Minikube v0.25+](https://github.com/kubernetes/minikube/releases/tag/v0.25.0)
* [Helm v2.8.2+](https://helm.sh)
* [Service Catalog CLI (svcat)](http://svc-cat.io/docs/install/#installing-the-service-catalog-cli)

## Setup

1. Create a Minikube cluster for local development: `make create-cluster`.
2. Identify where you will push Docker images with your changes and set the `REGISTRY`
   environment variable. For example to push your dev images to Docker Hub, use
   `export REGISTRY=myusername/`.

## Deploy

Compile and deploy the broker to your local cluster: `make push deploy`.

## Test

`make test`

Each of the tests is broken down into steps, so if you'd like to see what was
created before the testdata is removed just just the setup-* target, e.g. `make setup-mysql`.

There is an example chart for Wordpress that has been tweaked to use Minibroker for the
database provider, run `make setup-wordpress` to try it out.

## Dependency Management

We use [dep](https://golang.github.io/dep) to manage our dependencies. Our vendor
directory is checked-in and kept up-to-date with Gopkg.lock, so unless you are
actively changing dependencies, you don't need to do anything extra.

### Add a new dependency

1. Add the dependency.
    * Import the dependency in the code OR
    * Run `dep ensure --add github.com/pkg/example@v1.0.0` to add an explicit constraint
       to Gopkg.toml.

       This is only necessary when we want to stick with a particular branch
       or version range, otherwise the lock will keep us on the same version and track what's used.
2. Run `dep ensure`.
3. Check in the changes to `Gopkg.lock` and `vendor/`.
