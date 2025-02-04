#!/bin/bash
set -euo pipefail

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# Note: Avoid sourcing common.sh or common_version.sh in this script to allow
# its execution in a containerized environment with limited set of tools.

usage() {
    echo "Usage: $(basename "$0") download <version> <path>"
}

action_download() {
    local -r ver=$1
    local -r dir=$2

    if ! curl -I "https://brewhub.engineering.redhat.com" &>/dev/null ; then
        echo "ERROR: Brew Hub site is not accessible"
        exit 1
    fi

    # Attempt downloading current or previous release packages,
    # whatever latest build is available
    local package
    package=$(brew -q list-builds --package=microshift --state=COMPLETE | grep "^microshift-${ver}" | sort | tail -1) || true
    if [ -z "${package}" ] ; then
        echo "ERROR: Cannot find MicroShift '${ver}' packages in brew"
        exit 1
    fi

    package=$(awk '{print $1}' <<< "${package}")
    echo "Downloading '${package}' packages from brew"

    mkdir -p "${dir}"
    pushd "${dir}" &>/dev/null
    brew download-build --arch="$(uname -m)" "${package}"
    popd &>/dev/null
}

if [ $# -ne 3 ] ; then
    usage
    exit 1
fi
action="${1}"
shift

"${SCRIPTDIR}/../../scripts/fetch_tools.sh" brew

case "${action}" in
    download)
        "action_${action}" "$@"
        ;;
    -h)
        usage
        exit 0
        ;;
    *)
        usage
        exit 1
        ;;
esac
