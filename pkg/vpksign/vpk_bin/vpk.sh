#!/bin/bash
DIR=$(dirname "${BASH_SOURCE[0]}")
export LD_LIBRARY_PATH=$(cd "$DIR" && pwd)
exec "$DIR/vpk_linux32" "${@}"
