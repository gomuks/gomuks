#!/bin/sh
cd $(dirname "$0")
export MAU_GO_MOD_PATH=../../go.mod
export BINARY_NAME=libgomuksffi
export MAU_VERSION_PACKAGE=go.mau.fi/gomuks/version
export MAU_BUILD_CSHARED=true
export MAU_BUILD_PACKAGE_OVERRIDE=.
go tool maubuild "$@"
rm -f libgomuksffi.h
