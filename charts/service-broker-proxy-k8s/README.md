# service-broker-proxy-k8s

Kubernetes specific implementation for Service Broker Proxy module

## Introduction

This helm chart bootstraps the Service Broker Proxy for Kubernetes.

### Prerequisites

* `tiller` is installed and configured in the Kubernetes cluster.
* `helm` is installed and configured.
* `service-catalog` is installed and configured in the Kubernetes cluster.
* The Service Broker Proxy is registered in the Service Manager.

### Installation

```bash
helm install charts/service-broker-proxy-k8s --name service-broker-proxy --namespace service-broker-proxy --set config.sm.url=<SM_URL> --set sm.user=<USER> --set sm.password=<PASSWORD>
```

**Note:** Make sure you substitute &lt;SM_URL&gt; with the Service Manager url, &lt;USER&gt; and &lt;PASSWORD&gt; with the credentials for the Service Manager. The credentials can be obtained when registering the cluster in Service Manager.

To use your own images you can set `image.repository`, `image.tag` and `image.pullPolicy` to the helm install command.
