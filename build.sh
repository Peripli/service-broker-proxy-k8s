#!/usr/bin/env bash

REPOSITORY=$1
PROJECT=$2
VERSION=$3


docker build -f Dockerfile -t "$REPOSITORY"/"$PROJECT":"$VERSION"  .
docker push "$REPOSITORY"/"$PROJECT":"$VERSION"

cat yaml/service-broker-proxy-deployment.yaml \
    | sed -e "s/\${SERVICE_MANAGER_SECRET}/scp-broker/" \
    | sed -e "s/\${DOCKER_IMAGE_PULL_SECRET}/artifactory/" \
    | sed -e "s/\${REPOSITORY}/"$REPOSITORY"/" \
    | sed -e "s/\${PROJECT}/"$PROJECT"/" \
    | sed -e "s/\${VERSION}/"$VERSION"/" \
    | kubectl apply -f -
