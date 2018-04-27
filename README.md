# service-broker-proxy-k8s

[![Build Status](https://travis-ci.org/Peripli/service-broker-proxy-k8s.svg?branch=master)](https://travis-ci.org/Peripli/service-broker-proxy-k8s)

[![Coverage Status](https://coveralls.io/repos/github/Peripli/service-broker-proxy-k8s/badge.svg)](https://coveralls.io/github/Peripli/service-broker-proxy-k8s)

K8S Specific Implementation for Service Broker Proxy Module

## Docker Images

Docker Images are available on
https://console.cloud.google.com/gcr/images/gardener-project/EU/test/service-broker-proxy-k8s

## [WIP] Installation of the service broker proxy

Install [Helm](https://github.com/kubernetes/helm/releases)

```bash
$ kubectl -n kube-system create serviceaccount tiller
serviceaccount "tiller" created

$ kubectl create clusterrolebinding addon-tiller --clusterrole=cluster-admin --serviceaccount=kube-system:tiller
clusterrolebinding "addon-tiller" created

$ helm init --service-account tiller
```

Install service-catalog:

```bash
helm install charts/catalog --name catalog --namespace catalog
```

```bash
helm install charts/service-broker-proxy --name service-broker-proxy --namespace service-broker-proxy
# --set image.repository=fooo-repository # optional override of proxy's image
```

You can optionally create an override `values.yaml` file and override some of the default values:

```bash
helm install charts/service-broker-proxy \
  --name service-broker-proxy \
  --namespace service-broker-proxy \
  --values=charts/service-broker-proxy/values.yaml \
  --values=YOUR-OVERRIDE-FILE-HERE.yaml
```
