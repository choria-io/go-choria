#!/bin/bash

set -e

if [ -z $NAME ]
then
  NAME="choria"
fi

if [ -z $BINDIR ]
then
  BINDIR="/usr/sbin"
fi

if [ -z $ETCDIR ]
then
  ETCDIR="/etc/choria"
fi

if [ -z $VERSION ]
then
  echo "VERSION has not been set, cannot build"
  exit 1
fi

if [ -z $RELEASE ]
then
  RELEASE="1"
fi

if [ -z $DIST ]
then
  echo "DIST has not been set, cannot build"
  exit 1
fi

if [ -z $TARGET_DIST ]
then
  TARGET_DIST=${DIST}
fi

if [ -z $MANAGE_CONF ]
then
  MANAGE_CONF=1
fi

if [ -z $TARGET_ARCH ]
then
  TARGET_ARCH="x86_64"
fi

if [ -z $CONTACT ]
then
  CONTACT="R.I.Pienaar <rip@devco.net>"
fi

if [ ! -d /build ]
then
  echo "/build is not mounted, cannot build"
  exit 1
fi

if [ ! -d "/build/dist/${DIST}" ]
then
  echo "/build/dist/${DIST} is not mounted, cannot build"
  exit 1
fi

WORKDIR="${NAME}-${VERSION}"
BINARY="/build/choria-${VERSION}-linux-${TARGET_ARCH}"
TARBALL="${NAME}-${VERSION}-linux-${TARGET_ARCH}.tgz"

if [ ! -f ${BINARY} ]
then
  echo "${BINARY} does not exist, cannot build"
  exit 1
fi

mkdir -p ${WORKDIR}/dist

find /build/dist -maxdepth 1 -type f | xargs -I {} cp -v {} ${WORKDIR}/dist
cp -v /build/dist/${DIST}/* ${WORKDIR}/dist

for i in $(find ${WORKDIR}/dist -type f); do
  sed -i "s!{{pkgname}}!${NAME}!g" ${i}
  sed -i "s!{{bindir}}!${BINDIR}!g" ${i}
  sed -i "s!{{etcdir}}!${ETCDIR}!g" ${i}
  sed -i "s!{{version}}!${VERSION}!g" ${i}
  sed -i "s!{{iteration}}!${RELEASE}!g" ${i}
  sed -i "s!{{dist}}!${DIST}!g" ${i}
  sed -i "s!{{manage_conf}}!${MANAGE_CONF}!g" ${i}
  sed -i "s!{{target_dist}}!${TARGET_DIST}!g" ${i}
  sed -i "s!{{target_arch}}!${TARGET_ARCH}!g" ${i}
  sed -i "s!{{contact}}!${CONTACT}!g" ${i}
done

cp ${BINARY} ${WORKDIR}
tar -cvzf ${TARBALL} ${WORKDIR}

if [ ! -z $TARGET_ARCH ]
then
  rpmbuild --target "${TARGET_ARCH}" -ta "${TARBALL}"
else
  rpmbuild -ta "${TARBALL}"
fi

cp -v /usr/src/redhat/RPMS/*/* /build
cp -v /usr/src/redhat/SRPMS/* /build
