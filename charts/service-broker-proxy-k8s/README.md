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

Register the cluster in Service Manager as a platform resource.
This will generate and return credentials. For example:
```sh
smctl register-platform my-cluster kubernetes
```
**Note:** Store the returned credentials in a safe place as you will not be able to fetch them again from Service Manager.

From the root folder of this repository, execute:
```bash
helm install charts/service-broker-proxy-k8s \
  --name service-broker-proxy \
  --namespace service-broker-proxy \
  --set image.tag=<VERSION> \
  --set config.sm.url=<SM_URL> \
  --set sm.user=<USER> \
  --set sm.password=<PASSWORD>
```

**Note:** Make sure you substitute &lt;SM_URL&gt; with the Service Manager url, &lt;USER&gt; and &lt;PASSWORD&gt; with the credentials for the Service Manager.
Substitute \<VERSION> with the required version as listed on [Releases](https://github.com/Peripli/service-broker-proxy-k8s/releases). It is recommended to use the latest release.

To use your own images you can set `image.repository`, `image.tag` and `image.pullPolicy` to the helm install command. In case your image is pulled from a private repository, you can use
`image.pullsecret` to name a secret containing the credentials.
## Configuration

The following table lists some of the configurable parameters of the service broker proxy for K8S chart and their default values.

Parameter | Description | Default
--------- | ----------- | -------
`image.repository`| image repository |`quay.io/service-manager/sb-proxy-k8s`
`image.tag`| tag of image | `master`
`image.pullsecret` | name of the secret containing pull secrets |
`config.sm.url` | service manager url | `http://service-manager.dev.cfdev.sh`
`sm.user` | username for service manager | `admin`
`sm.password` | password for service manager | `admin`
`securityContext` | Custom [security context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/) for server containers | `{}`
