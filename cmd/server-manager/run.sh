#!/bin/bash
set -xe
export BUILD_TIME=$(date +'%s')
export DEBUG=true
export GO111MODULE=on
node_modules/.bin/babel javascript/manager.js -o static/manager.js
go build -ldflags "-s -w -X github.com/cj123/assetto-server-manager.BuildTime=${BUILD_TIME}"
./server-manager
