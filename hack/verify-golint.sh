#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

if ! which golint > /dev/null; then
    go get -u github.com/golang/lint/golint
fi

PKGS=$(go list qiniu.com/account/app/...)

ret=0
golint -set_exit_status ${PKGS[*]} || ret=$?

if [ $ret -eq 0 ]; then
    echo "Go source code passed golint check."
else
    exit $ret
fi
