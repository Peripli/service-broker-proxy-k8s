#!/usr/bin/env bash

########################################################################################################################
# JUST FOR LOCAL DEVELOPMENT and deployment to kubernetes. Not for CI/CD pipeline
########################################################################################################################
# usage:
# Call this script with the version to build and push to the registry. After build/push the
# yaml/* files are deployed into your cluster
#
#  ./build.sh 1.0
#

VERSION=$1
PROJECT=service-broker-proxy-k8s
REPOSITORY=cp-enablement.docker.repositories.sap.ondemand.com
NAMESPACE=service-broker-proxy

# causes the shell to exit if any subcommand or pipeline returns a non-zero status.
set -e
# set debug mode
#set -x


########################################################################################################################
# build the new docker image
########################################################################################################################
#
echo '>>> Building new image'
# Due to a bug in Docker we need to analyse the log to find out if build passed (see https://github.com/dotcloud/docker/issues/1875)
docker build --no-cache=true -t $REPOSITORY/$PROJECT:$VERSION . | tee /tmp/docker_build_result.log
RESULT=$(cat /tmp/docker_build_result.log | tail -n 1)
if [[ "$RESULT" != *Successfully* ]];
then
  exit -1
fi

########################################################################################################################
# push the docker image to your registry
########################################################################################################################
#
echo '>>> Push new image'
docker push $REPOSITORY/$PROJECT:$VERSION


########################################################################################################################
# deploy your YAML files into your kubernetes cluster via helm
########################################################################################################################

helm upgrade --install service-broker-proxy \
  charts/service-broker-proxy \
  --set image.repository=$REPOSITORY/$PROJECT \
  --set image.tag=$VERSION \
  --namespace $NAMESPACE