#!/bin/bash

export GOPATH=$(pwd)/third_party:~/go:$GOPATH
export PATH=$(pwd)/third_party/bin:$PATH

pushd proto
protoc --go_out=../tally/ tally.proto
protoc --go_out=../lighthouse/ lighthouse.proto
protoc --go_out=../passport/ passport.proto
popd
