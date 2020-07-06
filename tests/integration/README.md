# Integration tests

This is a suite of tests to assert Minibroker integration functionality.

## Requirements

You will need `ginkgo` installed. You can install it using the latest instructions from
https://onsi.github.io/ginkgo/#getting-ginkgo.

## Running the test suites

The tests assume the Service Catalog and Minibroker are already deployed. A namespace for deploying
the service instances is also required to be created ahead of time.

Environment variables:
- NAMESPACE (required): the namespace used by Minibroker to provision the service instances.
- TEST_BROKER_READY_TIMEOUT (optional): a timeout for waiting for the service broker to be ready.
- TEST_PROVISION_TIMEOUT (optional): a timeout for waiting for the provisioning to complete.
- TEST_BIND_TIMEOUT (optional): a timeout for waiting for the binding to complete.
- TEST_ASSERT_TIMEOUT (optional): a timeout for waiting for the assertion to complete.

An example running the tests:

```
kubectl create namespace minibroker-tests
NAMESPACE=minibroker-tests ginkgo --nodes 4 --slowSpecThreshold 180 .
```
