# cb-network (POC, Proof of Concept)

cb-network is under-study.
[TBD] Overview of cb-network

# Getting started with cb-network Server
## Prerequisites
<ins>**To be deprecated when `go module` is added**</ins>

### Install Golang 1.15.3
Please refer to [Go Setup Script](https://github.com/cb-contributhon/cb-coffeehouse/tree/master/scripts/go-setup)
```
wget https://raw.githubusercontent.com/cb-contributhon/cb-coffeehouse/master/scripts/go-setup/go1.15.3-setup.sh
source go1.15.3-setup.sh
```
### Get external packages 
```
go get -u github.com/eclipse/paho.mqtt.golang
go get -u github.com/labstack/echo
go get -u github.com/songgao/water
go get -u golang.org/x/net/ipv4
```

## How to run cb-network Server
### Get CB-Larva package
```
go get -u github.com/cloud-barista/cb-larva
```

### Change directory
```
cd $GOPATH/src/github.com/cloud-barista/cb-larva/poc-cb-net/cmd/server
```

### Build cb-network Server
```
go build server.go
```

### Run cb-network Server
```
sudo ./server
```


# Getting started with cb-network Agent
## Prerequisites
<ins>**To be deprecated when `go module` is added**</ins>

### Install Golang 1.15.3
**If you already install golang 1.15.3 in the above cb-network Server part, you can skip this.**
Please refer to [Go Setup Script](https://github.com/cb-contributhon/cb-coffeehouse/tree/master/scripts/go-setup)
```
wget https://raw.githubusercontent.com/cb-contributhon/cb-coffeehouse/master/scripts/go-setup/go1.15.3-setup.sh
source go1.15.3-setup.sh
```

### Get external packages 
**If you already install golang 1.15.3 in the above cb-network Server part, you can skip this.**
```
go get -u github.com/eclipse/paho.mqtt.golang
go get -u github.com/songgao/water
go get -u golang.org/x/net/ipv4
```

## How to run cb-network Agent
### Get CB-Larva package
```
go get -u github.com/cloud-barista/cb-larva
```

### Change directory
```
cd $GOPATH/src/github.com/cloud-barista/cb-larva/poc-cb-net/cmd/agent
```

### Build cb-network Agent
```
go build agent.go
```

### Run cb-network Agent
```
sudo ./agent
```
