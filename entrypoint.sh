#!/bin/sh
set -e

EXEC_BIN_PATH=${PGGAT_BIN_PATH:=/usr/bin/pggat}

pggat() {
    exec $EXEC_BIN_PATH ${@}
}


case "${1}" in
    "")
        pggat ${@}
        ;;
    *)
        export PGGAT_RUN_MODE=${1}
        shift
        pggat ${@}
        ;;
esac
