#!/usr/bin/env bash

# Example usage:
# ./deploy.sh eu.gcr.io gardener-project/test/service-broker-proxy-k8s 0.0.1-3f5b12c019af61a4ff90e09bf042ec331edf92df

REPOSITORY=$1
PROJECT=$2
VERSION=$3

cat yaml/service-broker-proxy-deployment.yaml \
    | sed -e "s#\${REPOSITORY}#"$REPOSITORY"#" \
    | sed -e "s#\${PROJECT}#"$PROJECT"#" \
    | sed -e "s#\${VERSION}#"$VERSION"#" \
    | kubectl apply -f -
