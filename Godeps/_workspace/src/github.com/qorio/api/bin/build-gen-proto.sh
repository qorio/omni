#!/bin/bash

source bin/env.sh

# Assumes GOPATH points to a single directory
export PATH=$GOPATH/bin:$PATH

pushd proto
protoc --go_out=../tally/ tally.proto
protoc --go_out=../passport/ passport.proto
protoc --go_out=../lighthouse/ lighthouse.proto
protoc --go_out=../soapbox/ soapbox.proto
popd
