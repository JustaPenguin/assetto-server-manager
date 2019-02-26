#!/bin/bash
set -xe
export BUILD_TIME=$(date +'%s')
export DEBUG=true
export GO111MODULE=on
go build -ldflags "-s -w -X github.com/cj123/assetto-server-manager.BuildTime=${BUILD_TIME}"
./server-manager
