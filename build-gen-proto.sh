#!/bin/bash

pushd proto
protoc --go_out=../tally/ tally.proto
popd

