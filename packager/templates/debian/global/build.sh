#!/bin/bash

set -x

TARGET_ARCH="{{cpkg_target_arch}}"
TARBALL="{{cpkg_tarball}}"
SOURCE="{{cpkg_source_dir}}"

cd {{cpkg_name}}-{{cpkg_version}}

mv dist debian
mv debian/server.service debian/{{cpkg_name}}.{{cpkg_name}}-server.service
mv debian/broker.service debian/{{cpkg_name}}.{{cpkg_name}}-broker.service

cp "${SOURCE}"/LICENSE debian/copyright

if [ ! -z "${TARGET_ARCH}" ]
then
  dpkg-buildpackage -us -uc --host-type ${TARGET_ARCH}
else
  dpkg-buildpackage -us -uc
fi
