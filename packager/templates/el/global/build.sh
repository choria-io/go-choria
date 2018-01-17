#!/bin/bash

set -x

TARGET_ARCH="{{cpkg_target_arch}}"
TARBALL="{{cpkg_tarball}}"

if [ ! -z $TARGET_ARCH ]
then
  rpmbuild --target "${TARGET_ARCH}" -ta "${TARBALL}"
else
  rpmbuild -ta "${TARBALL}"
fi
