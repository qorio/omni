#!/bin/bash

export GOPATH=$(pwd)/third_party:~/go
export PATH=$(pwd)/third_party/bin:$PATH

pushd proto
protoc --go_out=../tally/ tally.proto
protoc --go_out=../lighthouse/ lighthouse.proto
popd
