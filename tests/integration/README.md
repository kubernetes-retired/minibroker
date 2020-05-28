# Integration tests

This is a suite of tests to assert Minibroker integration functionality.

## Requirements

You will need `ginkgo` installed. You can install it using the latest instructions from
https://onsi.github.io/ginkgo/#getting-ginkgo.

## Running the test suites

The tests assume the Service Catalog and Minibroker are already deployed. A namespace for deploying
the service instances are also required to be created ahead of time.

An example running the tests:

```
kubectl create namespace minibroker-tests
NAMESPACE=minibroker-tests ginkgo --nodes 4 --slowSpecThreshold 180 .
```
