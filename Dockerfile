## This is a Dockerfile for cb-network server

##############################################################
## Stage 1 - Go Build
##############################################################

FROM golang:1.15.3 AS builder

ENV GO111MODULE=on

COPY . /cb-larva

WORKDIR /cb-larva/poc-cb-net/cmd/server/

# Build the server
# Note - Using cgo write normal Go code that imports a pseudo-package "C". I may not need on cross-compiling.
# Note - You can find possible platforms by 'go tool dist list' for GOOS and GOARCH
# Note - Using the -ldflags parameter can help set variable values at compile time.
# Note - Using the -s and -w linker flags can strip the debugging information.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s -w' -o server

#############################################################
## Stage 2 - Application Setup
##############################################################

FROM alpine:latest

WORKDIR /app

RUN mkdir -p configs
RUN mkdir -p web

# Copy the execution file
COPY --from=builder /cb-larva/poc-cb-net/cmd/server/server .
# Copy the web files of AdminWeb
COPY --from=builder /cb-larva/poc-cb-net/web/ ./web/

# A port of Admin Web for the cb-network controller
EXPOSE 9999

ENTRYPOINT ["./server"]
