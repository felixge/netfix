#!/usr/bin/env bash

# ./deploy.bash <host> creates and deploys a new build.
#
# Note: This is not intendet to work for you unless you're me. At least for now
# :).

set -eu
echo "-> build"
GOOS=linux GOARCH=arm make clean build
echo "-> rsync"
rsync -az --delete build/ ${1:-pi}:netfix
echo "-> install"
ssh pi "cd netfix && sudo ./pi_install.sh"
