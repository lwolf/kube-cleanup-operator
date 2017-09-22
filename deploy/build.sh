#!/bin/bash

set -e
set -x

deployDir=$(cd "$(dirname "$BASH_SOURCE")"; pwd)
source $deployDir/const.sh

deployFile="$deployDir/Dockerfile"

# If we don't have a build image for the latest code
if [[ ! $deployImage == *"snapshot"* ]] && [ -n "$(docker images --format "{{.Repository}}:{{.Tag}}" | grep $deployImage)" ] ; then
	echo "$deployImage already built"
	exit 0
fi

docker build -t $deployImage -f $deployFile $repoDir
