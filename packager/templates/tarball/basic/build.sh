#!/bin/sh

set -e

NAME="{{cpkg_name}}-{{cpkg_target_arch}}-{{cpkg_version}}"
DESTDIR="/tmp/choria/${NAME}"

mkdir -p $DESTDIR
mkdir /tmp/build

tar -C /tmp/build -xvzf "{{cpkg_tarball}}"

cp -v "/tmp/build/{{cpkg_name}}-{{cpkg_version}}/{{cpkg_binary}}" "${DESTDIR}/choria"
cd /tmp/choria
tar -cvzf "/tmp/choria/${NAME}.tgz" "${NAME}"
