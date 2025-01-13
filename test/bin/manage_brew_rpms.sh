#!/bin/bash
set -euo pipefail

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# shellcheck source=test/bin/common.sh
source "${SCRIPTDIR}/common.sh"
# shellcheck source=test/bin/common_versions.sh
source "${SCRIPTDIR}/common_versions.sh"

usage() {
    echo "Usage: $(basename "$0") <download | cleanup>"
}

action_download() {
    if ! which brew &>/dev/null ; then
        "${ROOTDIR}/scripts/fetch_tools.sh" brew
    fi

    if ! curl -I "https://brewhub.engineering.redhat.com" &>/dev/null ; then
        echo "ERROR: Brew Hub site is not accessible"
        exit 1
    fi

    # Attempt downloading current or previous release packages,
    # whatever latest build is available
    local got_package=false
    for ver in "${MINOR_VERSION}" "${PREVIOUS_MINOR_VERSION}" ; do
        local package
        package=$(brew -q list-builds --package=microshift --state=COMPLETE | grep "^microshift-4.${ver}" | sort | tail -1)
        if [ -z "${package}" ] ; then
            echo "WARNING: Cannot find MicroShift '4.${ver}' packages in brew. Skipping"
            continue
        fi

        package=$(awk '{print $1}' <<< "${package}")
        echo "Downloading '${package}' packages from brew"
        got_package=true

        mkdir -p "${BREW_REPO}"
        pushd "${BREW_REPO}" &>/dev/null
        brew download-build --arch="$(uname -m)" "${package}"
        popd &>/dev/null
        break
    done

    if ! ${got_package} ; then
        echo "ERROR: Could not find '4.${MINOR_VERSION}' or '4.${PREVIOUS_MINOR_VERSION}' packages in brew"
        exit 1
    fi
}

action_cleanup() {
    if [ -d "${BREW_REPO}" ] ; then
        rm -rfv "${BREW_REPO}"/*.rpm
    fi
}

if [ $# -ne 1 ]; then
    usage
    exit 1
fi
action="${1}"
shift

case "${action}" in
    download|cleanup)
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
