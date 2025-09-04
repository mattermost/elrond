#!/bin/bash
set -e
set -u

: ${GITHUB_REF_TYPE:?}
: ${GITHUB_REF_NAME:?}

# Note: This script is deprecated - CI now uses efficient multi-arch pattern
# For backwards compatibility, use old multi-arch build method
echo "⚠️  Using legacy multi-arch build - consider using the new CI workflow pattern"

if [ "${GITHUB_REF_TYPE:-}" = "branch" ]; then
  echo "Pushing latest for $GITHUB_REF_NAME..."
  export TAG=latest
else
  echo "Pushing release $GITHUB_REF_NAME..."
  export TAG="$GITHUB_REF_NAME"
fi
make build-image-with-tag
