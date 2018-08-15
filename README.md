# service-broker-proxy-k8s

[![Build Status](https://travis-ci.org/Peripli/service-broker-proxy-k8s.svg?branch=master)](https://travis-ci.org/Peripli/service-broker-proxy-k8s)[![Coverage Status](https://coveralls.io/repos/github/Peripli/service-broker-proxy-k8s/badge.svg?branch=master)](https://coveralls.io/github/Peripli/service-broker-proxy-k8s?branch=master)[![Go Report Card](https://goreportcard.com/badge/github.com/Peripli/service-broker-proxy-k8s)](https://goreportcard.com/report/github.com/Peripli/service-broker-proxy-k8s)

K8S Specific Implementation for Service Broker Proxy Module

## Docker Images

Docker Images are available on quay.io/service-manager/sb-proxy

## Installation of the service broker proxy on Kubernetes

### Prerequisites

* `tiller` is installed and configured in the Kubernetes cluster.
* `helm` is installed and configured.
* `service-catalog` is installed and configured in the Kubernetes cluster.

### Installation

The service-broker-proxy-k8s is installed via a helm chart.

```bash
helm install charts/service-broker-proxy --name service-broker-proxy --namespace service-broker-proxy --set config.sm.host=<SM_HOST> --set sm.user=<USER> --set sm.password=<PASSWORD>
```

**Note:** Make sure you substitute <SM_HOST> with the Service Manager url, <USER> and <PASSWORD> with the credentials for the Service Manager. The credentials can be obtained when registering the cluster in Service Manager.

To use your own images you can set `image.repository`, `image.tag` and `image.pullPolicy` to the helm install command.
