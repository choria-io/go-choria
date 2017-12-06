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

if [ -z $MANAGE_CONF ]
then
  MANAGE_CONF=1
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
BINARY="/build/choria-${VERSION}-Linux-amd64"
TARBALL="${NAME}-${VERSION}-Linux-amd64.tgz"

if [ ! -f ${BINARY} ]
then
  echo "${BINARY} does not exist, cannot build"
  exit 1
fi

mkdir -p ${WORKDIR}/dist
cp ${BINARY} ${WORKDIR}

find /build/dist -maxdepth 1 -type f | xargs -I {} cp -v {} ${WORKDIR}/dist
cp -v /build/dist/${DIST}/* ${WORKDIR}/dist

tar -cvzf ${TARBALL} ${WORKDIR}

rpmbuild \
  -D "version ${VERSION}" \
  -D "iteration ${RELEASE}"\
  -D "dist ${DIST}" \
  -D "pkgname ${NAME}" \
  -D "bindir ${BINDIR}" \
  -D "etcdir ${ETCDIR}" \
  -D "manage_conf ${MANAGE_CONF}" \
  -ta ${TARBALL}

cp -v ${TARBALL} /build
cp -v /usr/src/redhat/RPMS/x86_64/* /build
cp -v /usr/src/redhat/SRPMS/* /build
