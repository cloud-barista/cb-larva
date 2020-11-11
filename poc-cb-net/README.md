# CB-Network (POC, Proof of Concept)

CB-Network is under-study.
[TBD] Overview of CB-Network

# Getting started with CB-Network Server
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

## How to run CB-Network Server
### Get CB-Larva package
```
go get -u github.com/cloud-barista/cb-larva
```

### Change directory
```
cd $GOPATH/src/github.com/cloud-barista/cb-larva/poc-cb-net/cmd/server
```

### Build CB-Network Server
```
go build server.go
```

### Run CB-Network Server
```
sudo ./server
```


# Getting started with CB-Network Agent
## Prerequisites
<ins>**To be deprecated when `go module` is added**</ins>

### Install Golang 1.15.3
**If you already install golang 1.15.3 in the above CB-Network Server part, you can skip this.**
Please refer to [Go Setup Script](https://github.com/cb-contributhon/cb-coffeehouse/tree/master/scripts/go-setup)
```
wget https://raw.githubusercontent.com/cb-contributhon/cb-coffeehouse/master/scripts/go-setup/go1.15.3-setup.sh
source go1.15.3-setup.sh
```

### Get external packages 
**If you already install golang 1.15.3 in the above CB-Network Server part, you can skip this.**
```
go get -u github.com/eclipse/paho.mqtt.golang
go get -u github.com/songgao/water
go get -u golang.org/x/net/ipv4
```

## How to run CB-Network Agent
### Get CB-Larva package
```
go get -u github.com/cloud-barista/cb-larva
```

### Change directory
```
cd $GOPATH/src/github.com/cloud-barista/cb-larva/poc-cb-net/cmd/agent
```

### Build CB-Network Agent
```
go build agent.go
```

### Run CB-Network Agent
```
sudo ./agent
```
