#!/bin/sh

set -e

NAME="{{cpkg_name}}-{{cpkg_os}}-{{cpkg_target_arch}}-{{cpkg_version}}"
DESTDIR="/tmp/choria/${NAME}"
EXT=""

mkdir -p $DESTDIR
mkdir /tmp/build

tar -C /tmp/build -xvzf "{{cpkg_tarball}}"

if [ "${GOOS}" = "windows" ]
then
  EXT=".exe"
fi

cp -v "/tmp/build/{{cpkg_name}}-{{cpkg_version}}/{{cpkg_binary}}" "${DESTDIR}/choria${EXT}"
cd /tmp/choria

case "{{cpkg_format}}" in
  tgz)
    tar -cvzf "/tmp/choria/${NAME}.tgz" "${NAME}"
    ;;
  zip)
     zip -r "/tmp/choria/${NAME}.zip" "${NAME}"
    ;;
  *)
    echo "unsupported format {{cpkg_format}}, supports 'tgz' and 'zip'"
    exit 1
    ;;
esac
