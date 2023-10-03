#!/bin/sh
set -e

EXEC_BIN_PATH=${PGGAT_BIN_PATH:=/usr/bin/pggat}

pggat() {
    $EXEC_BIN_PATH ${@}
}

if ! [[ -z "${PGGAT_RUN_MODE}" ]]; then
    $EXEC_BIN_PATH run --adapter="gatfile" --config="/presets/${PGGAT_RUN_MODE:=default}.Caddyfile"
    exit 0
fi


case "${1}" in
    "")
        pggat ${@}
        ;;
    "pgbouncer")
        shift
        $EXEC_BIN_PATH pgbouncer ${@}
        ;;
esac
