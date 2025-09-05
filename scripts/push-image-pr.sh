#!/bin/bash
set -e
set -u

: ${GITHUB_SHA:?}

export TAG="${GITHUB_SHA:0:7}"

# Note: This script is deprecated - CI now uses efficient multi-arch pattern
# For backwards compatibility, use old multi-arch build method
echo "⚠️  Using legacy multi-arch build - consider using the new CI workflow pattern"
make build-image-with-tag
