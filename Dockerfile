## This is a Dockerfile for cb-network server

##############################################################
## Stage 1 - Go Build
##############################################################

FROM golang:1.15.3 AS builder

ENV GO111MODULE=on

COPY . /cb-larva

WORKDIR /cb-larva/poc-cb-net/cmd/server/

# RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s -w' -o server

#############################################################
## Stage 2 - Application Setup
##############################################################

FROM alpine:latest

WORKDIR /app

RUN mkdir configs

COPY --from=builder /cb-larva .

# A port of Admin Web for the cb-network controller
EXPOSE 9999

WORKDIR /app/poc-cb-net/cmd/server

ENTRYPOINT ["./server"]
