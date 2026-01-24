#!/bin/sh
cd $(dirname "$0")
export MAU_GO_MOD_PATH=../../go.mod
export BINARY_NAME=ffi
export MAU_VERSION_PACKAGE=go.mau.fi/gomuks/version
export MAU_BUILD_CSHARED=true
export MAU_BUILD_PACKAGE_OVERRIDE=.
go tool maubuild "$@"
rm -f ffi.h
mv ffi.a libgomuksffi.a 2>/dev/null
mv ffi.so libgomuksffi.so 2>/dev/null
mv ffi.dll libgomuksffi.dll 2>/dev/null
mv ffi.dylib libgomuksffi.dylib 2>/dev/null
