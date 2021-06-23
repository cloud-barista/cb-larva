![ubuntu-passing](https://img.shields.io/badge/ubuntu18.04-passing-success)

*Read this in other languages: [English](https://github.com/cloud-barista/cb-larva/blob/master/poc-cb-net/README.md), [한국어](https://github.com/cloud-barista/cb-larva/blob/master/poc-cb-net/README.KR.md)*

# Cloud-Barista Network

Cloud-Barista Network (cb-network) is under-study. 
It is <ins>**the global scale network that copes with the differences and variability of cloud networks (e.g., VPC, vNet) 
to link cloud infrastructures around the world.**</ins>

As the top-level concept, it will gradually expand by adding network-related technologies (e.g., Subnet, DNS, and Load balancer). 
It could be a virtual network for Cloud-Barista independent of the CSPs' network.

Under the big concept of cb-network, we are researching and developing Cloud Adaptive Network (CLADNet / cb-cladnet).

<p align="center">
  <img src="https://user-images.githubusercontent.com/7975459/122491196-8130fe00-d01e-11eb-881e-1d3d3a2aa0c4.png">
</p>


## Cloud Adaptive Network

Cloud Adaptive Network is an overlay network that <ins>**can be adaptable to various networks in multi-cloud.**</ins>

CLADNet could provide a logical group of nodes with the common network (e.g., Subnet) and related core functions. 
Simply, **CLADNet (cb-cladnet)** provides a common network for multiple VMs and supports communication between VMs.

### CLADNet's directions
- Adaptive: an adaptable network which is adaptive to different cloud networks from multiple cloud service providers (CSPs)
- Fault tolerant: a global fault-tolerant network that can operate even in issues of CSPs and regions 
- Lightweight: A lightweight network that minimizes host (VM) resource usage
- Handy: An easy-to-use network for users or programs running on the CLADNet

### CLADNet's structures
- Event-driven architecture: We have chosen an event-driven architecture based on distributed key-value store. 
                                It performs efficient workflows by meaningful change events in services. 
                                The events occur during data change, creation, and deletion (CUD).
  - Moving towards a Microservice Architecture (MSA)
- Mesh topology: We have chosen the mesh topology for the overlay network. 
                    It's needed to minimize the performance difference depending on the location of the intermediary node.
  - Plan to improve structure with Pluggable Interface to apply other protocols such as IPSec


## Getting started with cb-network
### Prerequisites
#### Install packages/tools
- `sudo apt update -y`
- `sudo apt dist-upgrade -y`
- `sudo apt install git -y`

#### Install Golang
Please refer to [Go Setup Script](https://github.com/cloud-barista/cb-coffeehouse/tree/master/scripts/golang)
```
wget https://raw.githubusercontent.com/cloud-barista/cb-coffeehouse/master/scripts/golang/go-installation.sh
source go-installation.sh
```

#### Clone CB-Larva repository
```
git clone https://github.com/cloud-barista/cb-larva.git
```

#### Deploy the distributed key-value store
The cb-network system requires a distributed key-value store. 
You must deploy at least a single-node cluster of the distributed key-value store.

Please, refer to links below:
- [etcd 3.5 - Run etcd clusters inside containers](https://etcd.io/docs/v3.5/op-guide/container/)
- [etcd 3.5 - Quickstart](https://etcd.io/docs/v3.5/quickstart/)
- [etcd 3.5 - Demo](https://etcd.io/docs/v3.5/demo/)


### How to run cb-network Server
It was deployed and tested on the "home" directory of Ubuntu 18.04. You can start from YOUR_PROJECT_DIRECTORY.

#### Prepare configs for cb-network server
##### config.yaml
- Create `config.yaml` (Use the provided `template)config.yaml`)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/configs
  cp template)config.yaml config.yaml
  ```
- <ins>**Edit the "xxxx" part **</ins> of `etcd_cluster` and `admin_web` in the text below
- The config.yaml template:
  ```
  # configs for the both cb-network controller and agent as follows:
  etcd_cluster:
    endpoints: [ "xxx.xxx.xxx:xxxx", "xxx.xxx.xxx:xxxx", "xxx.xxx.xxx:xxxx" ]
  
  # configs for the cb-network controller as follows:
  admin_web:
    host: "xxx"
    port: "xxx"
  
  # configs for the cb-network agent as follows:
  cb_network:
    cladnet_id: "xxxx"
    host_id: "xxxx"
  
  demo_app:
    is_run: false
  ```

##### log_conf.yaml
- Create `config.yaml` (Use the provided `template)log_conf.yaml`)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/configs
  cp template)log_conf.yaml log_conf.yaml
  ```
- Edit `cblog` > `loglevel` if necessary
- The log_conf.yaml template:
  ```
  #### Config for CB-Log Lib. ####
  
  cblog:
    ## true | false
    loopcheck: true # This temp method for development is busy wait. cf) cblogger.go:levelSetupLoop().
  
    ## debug | info | warn | error
    loglevel: debug # If loopcheck is true, You can set this online.
  
    ## true | false
    logfile: false
  
  ## Config for File Output ##
  logfileinfo:
    filename: ./log/cblogs.log
    #  filename: $CBLOG_ROOT/log/cblogs.log
    maxsize: 10 # megabytes
    maxbackups: 50
    maxage: 31 # days
  ```
#### Change directory
```
cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/server
```

#### Build cb-network server
In the building process, the required packages are automatically installed based on the "go module". (Go module is very useful, isn't it?)
```
go build server.go
```

#### Run cb-network server
```
sudo ./server
```


### How to run cb-network agent
It was deployed and tested on the "home" directory of Ubuntu 18.04. You can start from YOUR_PROJECT_DIRECTORY.

#### Prepare configs for cb-network server
##### config.yaml
- Create `config.yaml` (Use the provided `template)config.yaml`)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/configs
  cp template)config.yaml config.yaml
  ```
- <ins>**Edit the "xxxx" part **</ins> of `etcd_cluster` and `cb_network` in the text below
- The config.yaml template:
  ```
  # configs for the both cb-network controller and agent as follows:
  etcd_cluster:
    endpoints: [ "xxx.xxx.xxx:xxxx", "xxx.xxx.xxx:xxxx", "xxx.xxx.xxx:xxxx" ]
  
  # configs for the cb-network controller as follows:
  admin_web:
    host: "xxx"
    port: "xxx"
  
  # configs for the cb-network agent as follows:
  cb_network:
    cladnet_id: "xxxx"
    host_id: "xxxx"
  
  demo_app:
    is_run: false
  ```

##### log_conf.yaml
- Create `config.yaml` (Use the provided `template)log_conf.yaml`)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/configs
  cp template)log_conf.yaml log_conf.yaml
  ```
- Edit `cblog` > `loglevel` if necessary
- The log_conf.yaml template:
  ```
  #### Config for CB-Log Lib. ####
  
  cblog:
    ## true | false
    loopcheck: true # This temp method for development is busy wait. cf) cblogger.go:levelSetupLoop().
  
    ## debug | info | warn | error
    loglevel: debug # If loopcheck is true, You can set this online.
  
    ## true | false
    logfile: false
  
  ## Config for File Output ##
  logfileinfo:
    filename: ./log/cblogs.log
    #  filename: $CBLOG_ROOT/log/cblogs.log
    maxsize: 10 # megabytes
    maxbackups: 50
    maxage: 31 # days
  ```
#### Change directory
```
cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/agent
```

#### Build cb-network agent
In the building process, the required packages are automatically installed based on the "go module".
```
go build agent.go
```

#### Run cb-network agent
```
sudo ./agent
```