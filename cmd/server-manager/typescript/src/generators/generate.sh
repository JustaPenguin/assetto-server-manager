#!/bin/bash

export GO111MODULE=on

go run racecontrol.go >../models/RaceControl.ts
go run udp.go >../models/UDP.ts

