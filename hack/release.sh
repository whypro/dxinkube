#!/usr/bin/env bash

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

set -o errexit
set -o nounset
set -o pipefail

export REGISTRY=index-dev.qiniu.io/kelibrary

docker login -u "${DOCKER_USERNAME}" -p "${DOCKER_PASSWORD}" index-dev.qiniu.io

images=(
morse
)

if [ $# -gt 0 ]; then
    images=$@
fi

TRAVIS_TAG=${TRAVIS_TAG:-} # if unset, set it to empty string
#
# examples:
#
#   v0.2.0
#   v0.2.1-alpha.0
#   v0.2.1-alpha.1
#   v0.2.1-alpha.x
#   v0.2.1-beta.0
#   v0.2.1
#
echo "TRAVIS_TAG: $TRAVIS_TAG"
if [[ "${TRAVIS_TAG}" != "" ]]; then
    if [[ "${TRAVIS_TAG}" =~ ^(v[0-9]+\.[0-9]+\.[0-9]+)(-(alpha|beta)\.[a-z0-9]+)?$ ]]; then
        export VERSION="${BASH_REMATCH[0]}"
        echo "VERSION: $VERSION"
        for image in "${images[@]}"; do
            echo "Pushing image '${image}' with tags '${VERSION}' and 'latest' to '${REGISTRY}'."
            make ${image}-os-linux
            make -C build/${image} build push push-latest
        done
    else
        echo "Invalid TRAVIS_TAG: ${TRAVIS_TAG}"
        exit 1
    fi
else
    echo "Building dirty releases..."
    # Replace '+' with '-', because docker image tag does not support '+' char.
    export VERSION=$(./hack/version.sh | awk -F' ' '/^VERSION:/ {print $2}' | tr -s '+' '-')
    echo "VERSION: $VERSION"
    for image in "${images[@]}"; do
        echo "Pushing image '${image}' with tags '${VERSION}' to '${REGISTRY}'."
        make ${image}-os-linux
        make -C build/${image} build push
    done
fi
