# Minibroker

> A minibroker for your minikube!

Minibroker is an implementation of the [Open Service Broker API](https://openservicebrokerapi.org)
suited for local development and testing. Rather than provisioning services
from a cloud provider, Minibroker provisions services in containers on the cluster.

Minibroker uses the [Kubernetes Helm Charts](https://github.com/kubernetes/charts) 
its source of provisionable services.

## Status
This is still a work-in-progress, it's not usable yet. 😊

# Install

```
make deploy
```

# Use

```
make test
```

# Local Development

## Requirements

* Docker
* [Minikube v0.25+](https://github.com/kubernetes/minikube/releases/tag/v0.25.0)
* [Helm v2.8.2+](https://helm.sh)
* [Service Catalog CLI (svcat)](https://github.com/kubernetes-incubator/service-catalog/cmd/svcat)

On a Mac you will also need either VirtualBox installed,
or the [Minikube xhyve driver](https://github.com/kubernetes/minikube/blob/master/docs/drivers.md#xhyve-driver)
which uses the hypervisor that comes with Docker for Mac.

The default Minikube driver is virtualbox, to use xhyve specify it in
**~/.minikube/config/config.json**:

```json
{
    "vm-driver": "xhyve"
}
```

## Initial Setup

1. Edit the Makefile and change the IMAGE to something that you can push to.
1. Create a Minikube cluster for local development: `make create-cluster`.

## Deploy

Compile and deploy the broker to your local cluster: `make deploy`.

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
