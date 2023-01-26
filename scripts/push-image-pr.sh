#!/bin/bash
set -e
set -u

#env >> BASH_ENV
#cat BASH_ENV | while read line; do
#	export $line
#done

: ${GITHUB_SHA:?}

export TAG="${GITHUB_SHA:0:7}"

make build-image-with-tag

#rm BASH_ENV
