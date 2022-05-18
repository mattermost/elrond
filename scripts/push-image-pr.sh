#!/bin/bash
set -e
set -u

env >> BASH_ENV
cat BASH_ENV | while read line; do
	export $line
done

export TAG="${CIRCLE_SHA1:0:7}"

echo $DOCKER_PASSWORD | docker login --username $DOCKER_USERNAME --password-stdin
make build-image-with-tag

rm BASH_ENV
