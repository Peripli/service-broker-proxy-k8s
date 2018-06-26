# service-broker-proxy-k8s

[![Build Status](https://travis-ci.org/Peripli/service-broker-proxy-k8s.svg?branch=master)](https://travis-ci.org/Peripli/service-broker-proxy-k8s)[![Coverage Status](https://coveralls.io/repos/github/Peripli/service-broker-proxy-k8s/badge.svg?branch=master)](https://coveralls.io/github/Peripli/service-broker-proxy-k8s?branch=master)[![Go Report Card](https://goreportcard.com/badge/github.com/Peripli/service-broker-proxy-k8s)](https://goreportcard.com/report/github.com/Peripli/service-broker-proxy-k8s)

K8S Specific Implementation for Service Broker Proxy Module

## Docker Images

Docker Images are available on
https://console.cloud.google.com/gcr/images/gardener-project/EU/test/service-broker-proxy-k8s

## [WIP] Installation of the service broker proxy

The service-broker-proxy-k8s is installed via a helm chart.
We first have to install helm and can then install the service-catalog and the service-broker-proxy-k8s afterwards.

1. Install [Helm](https://github.com/kubernetes/helm/releases)
    ```bash
    $ kubectl -n kube-system create serviceaccount tiller
    serviceaccount "tiller" created

    $ kubectl create clusterrolebinding addon-tiller --clusterrole=cluster-admin --serviceaccount=kube-system:tiller
    clusterrolebinding "addon-tiller" created

    $ helm init --service-account tiller
    ```

2. Install service-catalog via helm:
    ```bash
    helm install charts/catalog --name catalog --namespace catalog
    ```

3. Install service-broker-proxy-k8s
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

    Here is an example how the configuraiton can look like to connect the service-broker-proxy-k8s to the service-manager.

    Please double check the [values.yaml](charts/service-broker-proxy/values.yaml) to see which are the default values and which properties you can override in addition.
    Soon, there will be a "latest" tag and better versioned docker images.

    Check our current [docker registry](https://console.cloud.google.com/gcr/images/gardener-project/EU/test/service-broker-proxy-k8s) for all available images.
    ```yaml
    image:
      tag: 0.0.1-7c799a78c734866bdd2702c6b2958d4eb3dda49f # Version of the service-broker-proxy-k8s

    config:
      serviceManager:
        host: https://my-service-manager-instance.cfapps.eu10.hana.ondemand.com # The host url of the service manager
        user: servicemanageruser # username to access the service-manager
        password: servicemanagerpassword # password to access the service-manager
    ```
