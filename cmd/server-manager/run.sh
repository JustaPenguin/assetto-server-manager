#!/bin/bash
set -xe
export DEBUG=true
export GO111MODULE=on
go generate ./...
go run *.go
