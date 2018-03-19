#!/bin/bash

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

set -o errexit
set -o nounset
set -o pipefail

if ! which dep >/dev/null; then
    platform=$(uname -s | tr A-Z a-z)
    echo "Installing dep v0.4.1..."
    tmpfile=$(mktemp)
    trap "rm $tmpfile && echo $tmpfile removed" EXIT
    wget https://github.com/golang/dep/releases/download/v0.3.1/dep-${platform}-amd64 -O $tmpfile
    mv $tmpfile ${GOPATH}/bin/dep
    chmod +x ${GOPATH}/bin/dep
fi

function retry_with_sleep() {
    local n=${1:-0}
    shift
    for i in $(seq 1 $n); do
        "$@" && return 0 || sleep $i
    done
    "$@"
}

retry_with_sleep 3 dep ensure -vendor-only -v
