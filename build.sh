#!/bin/bash

GO_VERSION=${TRAVIS_GO_VERSION:-1.12.4}
VERSION=${TRAVIS_TAG:-dev}

SRC_PATH=${1:-$(pwd)}
RELEASE_PATH=${2:-$(pwd)/release}

NAME="firefox-history-merger"
TARGETS="windows/386 windows/amd64 linux/386 linux/amd64 darwin/386 darwin/amd64"
LDFLAGS="-s -w -X main.version=$VERSION"

echo "GO_VERSION=$GO_VERSION"
echo "VERSION=$VERSION"
echo "SRC_PATH=$SRC_PATH"
echo "RELEASE_PATH=$RELEASE_PATH"
echo "NAME=$NAME"
echo "TARGETS=$TARGETS"
echo "LDFLAGS=$LDFLAGS"
echo

rm -rf ${RELEASE_PATH}/*
docker run --rm -i \
  -v "$SRC_PATH:/source" \
  -v "$RELEASE_PATH:/build/github.com/crazy-max" \
  -e "PACK=cmd" \
  -e "OUT=firefox-history-merger" \
  -e "TARGETS=${TARGETS}" \
  -e "FLAG_LDFLAGS=${LDFLAGS}" \
  -e "FLAG_V=true" \
  -e "FLAG_X=true" \
  -e "GO111MODULE=on" \
  crazymax/xgo:${GO_VERSION}
