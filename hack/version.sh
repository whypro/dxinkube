#!/usr/bin/env bash
#
# This script is used to generate version informations from git repo automatically.
#
# Output format:
#
#   <KEY1>: <VALUE1>
#   <KEY2>: <VALUE2>
#
# You can simply use this command to extrac the value by key:
#
#   ./hack/version.sh | awk -F' ' '/^VERSION:/ {print $2}'
#

set -o errexit
set -o nounset
set -o pipefail

ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
cd $ROOT

GIT_EXACT_TAG=$(git describe --tags --abbrev=0 --exact-match 2>/dev/null || true)
GIT_RECENT_TAG=$(git describe --tags --abbrev=0 2>/dev/null || true)
GIT_SHA_SHORT=$(git rev-parse --short HEAD)
GIT_DIRTY=$(test -n "`git status --porcelain`" && echo "dirty" || echo "clean")

echo "GIT_EXACT_TAG: $GIT_EXACT_TAG"
echo "GIT_RECENT_TAG: $GIT_RECENT_TAG"
echo "GIT_DIRTY: $GIT_DIRTY"
echo "GIT_SHA_SHORT: $GIT_SHA_SHORT"

# next automatically increment last identifier.
function next_tag() {
    local v=$1
    if [[ "${v}" =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)(-(alpha|beta)\.([0-9]+))?$ ]]; then
        local major=${BASH_REMATCH[1]}
        local minor=${BASH_REMATCH[2]}
        local patch=${BASH_REMATCH[3]}
        local preIdentifiers=${BASH_REMATCH[4]}
        local preIdentifiers1=${BASH_REMATCH[5]}
        local preIdentifiers2=${BASH_REMATCH[6]}
        if [[ "${preIdentifiers}" != "" ]]; then
            preIdentifiers2=$(($preIdentifiers2 + 1))
        else
            patch=$(($patch + 1))
            preIdentifiers1="alpha"
            preIdentifiers2="0"
        fi
        echo "v${major}.${minor}.${patch}-${preIdentifiers1}.${preIdentifiers2}"
    else
        echo "unsupport version: $v"
        exit 1
    fi
}

VERSION=""
if [[ "$GIT_EXACT_TAG" != "" ]]; then
	VERSION=$GIT_EXACT_TAG
else
    GIT_NEXT_TAG=$(next_tag "$GIT_RECENT_TAG")
	if [[ "$GIT_DIRTY" == "dirty" ]]; then
		VERSION="$GIT_NEXT_TAG+dirty.$GIT_SHA_SHORT"
	else
		VERSION="$GIT_NEXT_TAG+$GIT_SHA_SHORT"
	fi
fi
echo "VERSION: $VERSION"
