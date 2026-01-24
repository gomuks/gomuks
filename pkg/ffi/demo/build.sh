#!/bin/sh
cd $(dirname $0)
../build.sh
gcc -Wall -Wextra -o demo demo.c -L.. -lgomuksffi -Wl,-rpath,'$ORIGIN'
echo
echo 'Run with LD_LIBRARY_PATH=..:$LD_LIBRARY_PATH GOMUKS_ROOT=$HOME/gomuksffidemo ./demo'
