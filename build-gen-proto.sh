#!/bin/bash

export GOPATH=$(pwd)/third_party:~/go:$GOPATH
export PATH=$(pwd)/third_party/bin:$PATH

pushd proto
protoc --go_out=../tally/ tally.proto
popd


pushd imports/src/github.com/qorio/api/proto
protoc --go_out=../lighthouse lighthouse.proto
protoc --go_out=../passport passport.proto
protoc --go_out=../soapbox soapbox.proto
popd
