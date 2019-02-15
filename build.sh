#!/bin/bash

GO_VERSION=${TRAVIS_GO_VERSION:-1.11.x}
VERSION=${1:-dev}
SRC_PATH=${2:-$(pwd)}
RELEASE_PATH=${3:-$(pwd)/release}

NAME="firefox-history-merger"
PACKAGE="github.com/crazy-max/firefox-history-merger"
TARGETS="windows/386 windows/amd64 linux/386 linux/amd64 darwin/386 darwin/amd64"
LDFLAGS="-s -w -X '$PACKAGE/utils.AppVersion=$VERSION'"

echo "VERSION=$VERSION"
echo "SRC_PATH=$SRC_PATH"
echo "RELEASE_PATH=$RELEASE_PATH"
echo "NAME=$NAME"
echo "PACKAGE=$PACKAGE"
echo "TARGETS=$TARGETS"
echo "LDFLAGS=$LDFLAGS"
echo

rm -rf ${RELEASE_PATH}/*
docker run --rm -i \
  -v "$SRC_PATH:/source" \
  -v "$RELEASE_PATH:/build/github.com/crazy-max" \
  -e "TARGETS=${TARGETS}" \
  -e "FLAG_LDFLAGS=${LDFLAGS}" \
  -e "FLAG_V=false" \
  -e "FLAG_X=false" \
  -e "GO111MODULE=on" \
  crazymax/xgo:${GO_VERSION}

mv ${RELEASE_PATH}/${NAME}-darwin-10.6-386 ${RELEASE_PATH}/${NAME}-${VERSION}-darwin-10.6-386
mv ${RELEASE_PATH}/${NAME}-darwin-10.6-amd64 ${RELEASE_PATH}/${NAME}-${VERSION}-darwin-10.6-amd64
mv ${RELEASE_PATH}/${NAME}-linux-386 ${RELEASE_PATH}/${NAME}-${VERSION}-linux-386
mv ${RELEASE_PATH}/${NAME}-linux-amd64 ${RELEASE_PATH}/${NAME}-${VERSION}-linux-amd64
mv ${RELEASE_PATH}/${NAME}-windows-4.0-386.exe ${RELEASE_PATH}/${NAME}-${VERSION}-windows-4.0-386.exe
mv ${RELEASE_PATH}/${NAME}-windows-4.0-amd64.exe ${RELEASE_PATH}/${NAME}-${VERSION}-windows-4.0-amd64.exe
