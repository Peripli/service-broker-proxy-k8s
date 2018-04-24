#!/usr/bin/env bash

# Example usage:
# ./build.sh cp-enablement.docker.repositories.sap.ondemand.com service-broker-proxy dev

REPOSITORY=$1
PROJECT=$2
VERSION=$3

docker build  -t "$REPOSITORY"/"$PROJECT":"$VERSION"  .
#docker push "$REPOSITORY"/"$PROJECT":"$VERSION"

exit 1
cat yaml/service-broker-proxy-deployment.yaml \
    | sed -e "s#\${REPOSITORY}#"$REPOSITORY"#" \
    | sed -e "s#\${PROJECT}#"$PROJECT"#" \
    | sed -e "s#\${VERSION}#"$VERSION"#" \
    | kubectl apply -f -
