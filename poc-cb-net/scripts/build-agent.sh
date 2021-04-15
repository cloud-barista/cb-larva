#!/bin/bash

# Change to the directory where agent.go is located
cd ../cmd/agent

# Build agent
# Note - Using the -ldflags parameter can help set variable values at compile time.
# Note - Using the -s and -w linker flags can strip the debugging information.
go build -mod=mod -a -ldflags '-s -w' -o agent