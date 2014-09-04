#!/bin/bash

export GOPATH=$(pwd)/third_party:~/go
export PATH=$(pwd)/third_party/bin:$PATH

# Run go oracle for development https://godoc.org/code.google.com/p/go.tools/oracle
if [[ $(which oracle) == "" ]]; then
    echo "Setting up go oracle for source code analysis."
    go install code.google.com/p/go.tools/cmd/oracle
fi

if [[ $(which godoc) == "" ]]; then
    echo "Godoc not installed."
fi
