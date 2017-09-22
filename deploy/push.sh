#!/bin/bash

set -e
set -x

deployDir=$(cd "$(dirname "$BASH_SOURCE")"; pwd)
source $deployDir/const.sh

# Ensure we've built the latest code
$deployDir/build.sh

latestImage=$repo:latest
docker tag $deployImage $registry/$deployImage
gcloud docker -- push $registry/$deployImage
docker tag $deployImage $registry/$latestImage
gcloud docker -- push $registry/$latestImage
