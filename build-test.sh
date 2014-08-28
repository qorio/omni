#!/bin/bash

WORKING=$(pwd)
GOPATH=$WORKING/imports:$WORKING/third_party:$GOPATH

TARGETS=""
for t in $@; do
    TARGETS="github.com/qorio/omni/$t $TARGETS"
done

echo "Targets are $TARGETS"
go test $TARGETS -v --logtostderr --auth_public_key_file=$WORKING/test/authKey.pub