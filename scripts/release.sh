# Usage: sh release.sh
# Note: To run this script locally you need to export environment variables GITHUB_REF and GH_TOKEN.
# Stop script on first error
set -xe

# Verify that required variables are available
: ${GITHUB_REF:?}
: ${GH_TOKEN:?}

echo $PATH
find / -name release-notes -type f

export GITHUB_TOKEN=${GH_TOKEN}

# Reference on GITHUB_REF format: https://docs.github.com/en/actions/learn-github-actions/variables
REF=$(echo -n $GITHUB_REF | rev | cut -d/ -f1 | rev)

FUTURE_RELEASE_SHA=$(hub rev-parse HEAD)
LATEST_RELEASE=$(hub release | head -n 1)
# The following fixes the script to work in the case that there isn't a previous release in place.
if [[ "$LATEST_RELEASE" == "" ]]; then
  LATEST_RELEASE=$(git log --pretty=%H | tail -n 1)
fi
LATEST_RELEASE_SHA=$(hub rev-parse "${LATEST_RELEASE}")
LATEST_RELEASE_NEXT_COMMIT_SHA=$(hub log "${LATEST_RELEASE}"..HEAD --oneline --pretty=%H | tail -n1)
mkdir -p ./build/_output/docs/
release-notes --org mattermost --repo elrond --start-sha "${LATEST_RELEASE_NEXT_COMMIT_SHA}" --end-sha "${FUTURE_RELEASE_SHA}"  --output ./build/_output/docs/relnote.md --required-author "" --branch main
cat ./build/_output/docs/relnote.md | sed '/docs.k8s.io/ d' | sed -e "s/\# Release notes for/${REF}/g" | sed -e "s/Changelog since/Changelog since ${LATEST_RELEASE}/g"> ./build/_output/docs/relnote_parsed.md
echo "\n" >> ./build/_output/docs/relnote_parsed.md
echo "_Thanks to all our contributors!_" >> ./build/_output/docs/relnote_parsed.md
mv ./build/_output/docs/relnote_parsed.md ./build/_output/docs/relnote.md

make binaries

echo hub release create -d -F ./build/_output/docs/relnote.md -a ./build/_output/bin/elrond-linux-arm64 -a ./build/_output/bin/elrond-linux-amd64 -a ./build/_output/bin/elrond-darwin-amd64 -a ./build/_output/bin/elrond-darwin-arm64 ${REF}
rm -rf ./build/_output/
