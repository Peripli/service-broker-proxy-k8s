# service-broker-proxy-k8s

[![Build Status](https://travis-ci.org/Peripli/service-broker-proxy-k8s.svg?branch=master)](https://travis-ci.org/Peripli/service-broker-proxy-k8s)

[![Coverage Status](https://coveralls.io/repos/github/Peripli/service-broker-proxy-k8s/badge.svg)](https://coveralls.io/github/Peripli/service-broker-proxy-k8s)

K8S Specific Implementation for Service Broker Proxy Module

## [WIP] Installation of the service broker proxy
Create a configuration file *proxy.properties* and create a configmap from these properties with the name *service-broker-proxy-configuration*. Replace the service manager credentials and configurations accordingly.

```
namespace=<namespace>
service_manager_timeout=<timeout_in_seconds>
```

```sh
kubectl create configmap service-broker-proxy-configuration --from-env-file=proxy.properties
```

Then create a secret *service-manager-secret* containing the coordinates of the the service manager.
*url*, *user*, and *password* are base64 encoded.
```yaml
cat <<EOF | kubectl create -f -
apiVersion: v1
kind: Secret
metadata:
  name: service-manager-secret
type: Opaque
data:
  url: <base64-encoded-url>
  user: <base64-encoded-user>
  password: <base64-encoded-password>
EOF
```

