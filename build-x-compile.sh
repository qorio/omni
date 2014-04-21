#!/bin/bash

WORKING=$(pwd)

BUILD=/tmp/build

GITHUB=github.com/qorio/omni
DIRS_TO_COPY=$(ls -d */)
FILES_TO_COPY="GeoLiteCity.dat"

# Git commit hash / message
GIT_COMMIT_HASH=$(git rev-list --max-count=1 --reverse HEAD)
GIT_COMMIT_MESSAGE=$(git log -1 | tail -1 | sed -e "s/^[ ]*//g")
BUILD_TIMESTAMP=$(date +"%Y-%m-%d-%H:%M")

echo "Git commit $GIT_COMMIT_HASH ($GIT_COMMIT_MESSAGE) on $BUILD_TIMESTAMP"
sed -ri "s/@@GIT_COMMIT_HASH@@/${GIT_COMMIT_HASH}/g" runtime/build_info.go
sed -ri "s/@@GIT_COMMIT_MESSAGE@@/${GIT_COMMIT_MESSAGE}/g" runtime/build_info.go
sed -ri "s/@@BUILD_TIMESTAMP@@/${BUILD_TIMESTAMP}/g" runtime/build_info.go
sed -ri "s/@@BUILD_NUMBER@@/${CIRCLE_BUILD_NUM}/g" runtime/build_info.go

cat runtime/build_info.go

# Usage: ./build-x-compile.sh src/*.go

# Script for cross-compiling go binaries for different platforms
# SKIPPING darwin/amd64 -- that seems to create just some binary file that we can't tell.
# PLATFORMS="darwin/386 freebsd/386 freebsd/amd64 freebsd/arm linux/386 linux/amd64 linux/arm windows/386 windows/amd64"
PLATFORMS="darwin/386 linux/amd64 windows/amd64"

eval "$(go env)"


# Input source code .go files to build
SRC=$@

echo "Cleaning build directory $BUILD"
rm -rf $BUILD
mkdir -p $BUILD/target

GOPATH=$WORKING/third_party:$BUILD:$GOPATH

ROOTDIR=$BUILD/src/$GITHUB
mkdir -p $ROOTDIR


for dir in $DIRS_TO_COPY; do
    t=`echo $dir | sed -e 's/\///g'`
    echo "Copying $t to $ROOTDIR"
    cp -r $t $ROOTDIR
done

for f in $FILES_TO_COPY; do
    cp $f $ROOTDIR
done

echo "GOPATH=$GOPATH"
echo "ROOTDIR=$ROOTDIR"
echo "BUILD=$BUILD"
find $BUILD


for PLATFORM in $PLATFORMS; do
	export GOOS=${PLATFORM%/*}
	export GOARCH=${PLATFORM#*/}
	TARGET=$BUILD/target/${GOOS}_${GOARCH}
	echo "Building for ${GOOS} on ${GOARCH}: ${TARGET}"
	mkdir -p ${TARGET}
	pushd ${TARGET}
	for s in $SRC; do
		go build $ROOTDIR/$s
	done
	popd
done

# Make sure the GOOS/GOARCH environment variables are set correctly
eval "$(go env)"

echo "Finished. Built artifacts are in 'target':"

BINARIES=$(find $BUILD/target)
for i in $BINARIES; do
    if [ -f $i ]; then
	echo "Generated binary - $(file $i)"
    fi;
done
