#!/bin/bash

WORKING=$(pwd)
export GOPATH=$WORKING/imports:$WORKING/third_party:$GOPATH

# Generate protos
$WORKING/build-gen-proto.sh

TARGETS=""
for t in $@; do
    option=$(echo $t | grep -e '^-')
    if [[ "$option" == "" ]]; then
	TARGETS="github.com/qorio/omni/$t $TARGETS"
    else
	OPTIONS="$t $OPTIONS"
    fi
done

echo "Targets are $TARGETS with options $OPTIONS"
go test $TARGETS -v --logtostderr --auth_public_key_file=$WORKING/test/authKey.pub $OPTIONS
