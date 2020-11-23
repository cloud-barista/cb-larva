![ubuntu-passing](https://img.shields.io/badge/ubuntu18.04-passing-success)

*Read this in other languages: [English](https://github.com/cloud-barista/cb-larva/blob/master/poc-cb-net/README.md), [한국어](https://github.com/cloud-barista/cb-larva/blob/master/poc-cb-net/README.KR.md)*

# cb-network

cb-network is under-study. Proof of concept (POC) of **cb-subnet** is in progress.

## Overview of cb-network
cb-network is Global Network Service in Cloud-Barista. The objective of cb-network is <ins>**to provide a unified and efficient global network service on Multiple CSPs' heterogeneous network.**</ins>   
We hope to make cb-network independent from CSP's network.   
cb-network may include cb-subnet, cb-dns, and cb-loadbalancer. Further items are welcome.

<p align="center">
  <img src="https://user-images.githubusercontent.com/7975459/99206719-7ea7c500-27ff-11eb-96f3-bc912bf7143a.png">
</p>

Currently, we are focusing on R&D for **cb-subnet** among cb-network components. cb-dns and cb-loadbalancer will be added.
**cb-subnet** performs creating a common network for multiple VMs and supports communication between VMs.

## Getting started with cb-network Server
### Prerequisites
#### Install Golang 1.15.3
Please refer to [Go Setup Script](https://github.com/cb-contributhon/cb-coffeehouse/tree/master/scripts/go-setup)
```
wget https://raw.githubusercontent.com/cb-contributhon/cb-coffeehouse/master/scripts/go-setup/go1.15.3-setup.sh
source go1.15.3-setup.sh
```

### How to run cb-network Server
It was deployed and tested on the "home" directory of Ubuntu 18.04. You can start from YOUR_PROJECT_DIRECTORY.

#### Clone CB-Larva repository
```
git clone https://github.com/cloud-barista/cb-larva.git
```

#### Change directory
```
cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/server
```

#### Build cb-network Server
On building process, the required packages are automatically installed based on "go module". (Go module is very useful, isn't it?)
```
go build server.go
```

#### Run cb-network Server
```
sudo ./server
```


## Getting started with cb-network Agent
### Prerequisites
#### Install Golang 1.15.3
**If you already install golang 1.15.3 in the above cb-network Server part, you can skip this.**
Please refer to [Go Setup Script](https://github.com/cb-contributhon/cb-coffeehouse/tree/master/scripts/go-setup)
```
wget https://raw.githubusercontent.com/cb-contributhon/cb-coffeehouse/master/scripts/go-setup/go1.15.3-setup.sh
source go1.15.3-setup.sh
```

### How to run cb-network Agent
It was deployed and tested on the "home" directory of Ubuntu 18.04. You can start from YOUR_PROJECT_DIRECTORY.

#### Clone CB-Larva repository
```
git clone https://github.com/cloud-barista/cb-larva.git
```

#### Change directory
```
cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/agent
```

#### Build cb-network Agent
On building process, the required packages are automatically installed based on "go module". (Go module is very useful, isn't it?)

```
go build agent.go
```

#### Run cb-network Agent
```
sudo ./agent
```
