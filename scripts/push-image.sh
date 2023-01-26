#!/bin/bash
set -e
set -u

: ${GITHUB_REF_TYPE:?}
: ${GITHUB_REF:?}

#env >> BASH_ENV
#cat BASH_ENV | while read line; do
#	export $line
#done
if [ "${GITHUB_REF_TYPE:-}" = "branch" ]; then
  echo "Pushing latest for $GITHUB_REF..."
  export TAG=latest
else
  echo "Pushing release $GITHUB_REF..."
  export TAG="$GITHUB_REF"
fi
make build-image-with-tag

#rm BASH_ENV
