#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

ret=0
make vet || ret=$?

if [ $ret -eq 0 ]; then
    echo "Go source code passed vet check."
else
    echo "Go source code have suspicious constructs, please check error logs and refer docs at https://golang.org/cmd/vet/."
    exit $ret
fi
