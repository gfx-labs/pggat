#!/bin/sh
set -e

EXEC_BIN_PATH=${PGGAT_BIN_PATH:=/usr/bin/pggat}

$EXEC_BIN_PATH run --adapter="caddyfile" --config="/presets/${PGGAT_RUN_MODE:=default}.Caddyfile"
