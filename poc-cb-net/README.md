![ubuntu-passing](https://img.shields.io/badge/ubuntu18.04-passing-success)

*Read this in other languages: [English](https://github.com/cloud-barista/cb-larva/blob/master/poc-cb-net/README.md), [한국어](https://github.com/cloud-barista/cb-larva/blob/master/poc-cb-net/README.KR.md)*

**[Shortcut]**
- [An overview of Cloud-Barista Network](#an-overview-of-cloud-barista-network)
- [Introduction to Cloud Adaptive Network](#introduction-to-cloud-adaptive-network)
- [Getting started with cb-network system](#getting-started-with-cb-network-system)
  - [How to run a cb-network controller based on source code](#how-to-run-a-cb-network-controller-based-on-source-code)
  - [How to run a cladnet service based on source code](#how-to-run-a-cladnet-service-based-on-source-code)
  - [How to run an admin-web based on source code](#how-to-run-an-admin-web-based-on-source-code)
  - [How to run a cb-network agent based on source code](#how-to-run-a-cb-network-agent-based-on-source-code)
- [Demo: 1st step, to run existing services in multi-cloud](#demo-1st-step-to-run-existing-services-in-multi-cloud)


## An overview of Cloud-Barista Network

Cloud-Barista Network (cb-network) is under-study. 
It is <ins>**the global scale network that copes with the differences and variability of cloud networks (e.g., VPC, vNet) 
to link cloud infrastructures around the world.**</ins>

As the top-level concept, it will gradually expand by adding network-related technologies (e.g., Subnet, DNS, and Load balancer). 
It could be a virtual network for Cloud-Barista independent of the CSPs' network.

The cb-network will mainly represent systems or visions, and Cloud Adaptive Network (CLADNet) represent a technology under research and development.

<p align="center">
  <img src="https://user-images.githubusercontent.com/7975459/145977420-1e5af8b1-bf87-4282-917c-9c982915c332.png">
</p>


## Introduction to Cloud Adaptive Network

Cloud Adaptive Network (CLADNet) is simply an overlay network that <ins>**can be adaptable to various networks in multi-cloud.**</ins>

CLADNet could provide a logical group of nodes with the common network (e.g., Subnet) and related core functions. 
Simply, **CLADNet (cb-cladnet)** provides a common network for multiple VMs and supports communication between VMs.

<p align="center">
  <img src="https://user-images.githubusercontent.com/7975459/122491196-8130fe00-d01e-11eb-881e-1d3d3a2aa0c4.png">
</p>

### CLADNet's directions
- **Adaptive**: an adaptable network which is adaptive to different cloud networks from multiple cloud service providers (CSPs)
- **Fault tolerant**: a global fault-tolerant network that can operate even in issues of CSPs and regions 
- **Lightweight**: A lightweight network that minimizes host (VM) resource usage
- **Handy**: An easy-to-use network for users or programs running on the CLADNet

### CLADNet's structures
- Event-driven architecture: We have chosen an event-driven architecture based on distributed key-value store. 
                                It performs efficient workflows by meaningful change events in services. 
                                The events occur during data change, creation, and deletion (CUD).
  - Moving towards a Microservice Architecture (MSA)
- Mesh topology: We have chosen the mesh topology for the overlay network. 
                    It's needed to minimize the performance difference depending on the location of the intermediary node.
  - Research in progress to improve communication performance


## Getting started with cb-network system
This section describes the preparations required to start the cb-network system and how to run each component.
`cb-network controller`, `cb-network cladnet-service`, `cb-network admin-web`, and `distributed key-value store` can be run on the same node,
Each `cb-network agent` must be run on a different host (VM).
### Prerequisites
#### Install packages/tools
```
sudo apt update -y
sudo apt dist-upgrade -y
sudo apt install git -y
```

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
`etcd` is used, and a single-node cluster of etcd is deployed for testing.

Please, refer to links below:
- [etcd 3.5 - Run etcd clusters inside containers](https://etcd.io/docs/v3.5/op-guide/container/)
- [etcd 3.5 - Quickstart](https://etcd.io/docs/v3.5/quickstart/)
- [etcd 3.5 - Demo](https://etcd.io/docs/v3.5/demo/)

---

### How to run a cb-network controller based on source code
It was deployed and tested on the "home" directory of Ubuntu 18.04. You can start from YOUR_PROJECT_DIRECTORY.

#### Prepare the config for cb-network controller
##### config.yaml
- Create `config.yaml` (Use the provided `template-config.yaml`)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/config
  cp template-config.yaml config.yaml
  ```
- <ins>**Edit the "xxxx" part**</ins> of `etcd_cluster` in the text below
- The config.yaml template:
  ```
  # A config for the both cb-network controller and agent as follows:
  etcd_cluster:
    endpoints: [ "xxx.xxx.xxx.xxx:xxx", "xxx.xxx.xxx.xxx:xxx", "xxx.xxx.xxx.xxx:xxx" ]

  # A config for the cb-network AdminWeb as follows:
  admin_web:
    host: "localhost"
    port: "9999"

  # A config for the cb-network agent as follows:
  cb_network:
    cladnet_id: "xxxx"
    host_id: "" # if host_id is "" (empty string), the cb-network agent will use hostname.

  # A config for the grpc as follows:
  grpc:
    service_endpoint: "localhost:8089"
    server_port: "8089"
    gateway_port: "8088"

  ```

##### log_conf.yaml
- Create `config.yaml` (Use the provided `template-log_conf.yaml`)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/config
  cp template-log_conf.yaml log_conf.yaml
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
cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/controller
```

#### Build cb-network controller
In the building process, the required packages are automatically installed based on the "go module". (Go module is very useful, isn't it?)
```
go build controller.go
```

#### Run cb-network controller
```
sudo ./controller
```

---

### How to run a cladnet service based on source code
It was deployed and tested on the "home" directory of Ubuntu 18.04. You can start from YOUR_PROJECT_DIRECTORY.

#### Prepare the config for the cladnet service
##### config.yaml
- Create `config.yaml` (Use the provided `template-config.yaml`)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/config
  cp template-config.yaml config.yaml
  ```
- <ins>**Edit the "xxxx" part**</ins> of `etcd_cluster` and `grpc` in the text below
- The config.yaml template:
  ```
  # A config for the both cb-network controller and agent as follows:
  etcd_cluster:
    endpoints: [ "xxx.xxx.xxx.xxx:xxx", "xxx.xxx.xxx.xxx:xxx", "xxx.xxx.xxx.xxx:xxx" ]

  # A config for the cb-network AdminWeb as follows:
  admin_web:
    host: "localhost"
    port: "9999"

  # A config for the cb-network agent as follows:
  cb_network:
    cladnet_id: "xxxx"
    host_id: "" # if host_id is "" (empty string), the cb-network agent will use hostname.

  # A config for the grpc as follows:
  grpc:
    service_endpoint: "localhost:8089"
    server_port: "8089"
    gateway_port: "8088"

  ```

##### log_conf.yaml
- Create `config.yaml` (Use the provided `template-log_conf.yaml`)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/config
  cp template-log_conf.yaml log_conf.yaml
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
cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/service
```

#### Build the cladnet service
In the building process, the required packages are automatically installed based on the "go module".
```
go build cladnet-service.go
```

#### Run the cladnet service
```
sudo ./cladnet-service
```

---

### How to run an admin-web based on source code
It was deployed and tested on the "home" directory of Ubuntu 18.04. You can start from YOUR_PROJECT_DIRECTORY.

#### Prepare the config for the admin-web
##### config.yaml
- Create `config.yaml` (Use the provided `template-config.yaml`)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/config
  cp template-config.yaml config.yaml
  ```
- <ins>**Edit the "xxxx" part**</ins> of `etcd_cluster`, `admin_web`, and `grpc` in the text below
- The config.yaml template:
  ```
  # A config for the both cb-network controller and agent as follows:
  etcd_cluster:
    endpoints: [ "xxx.xxx.xxx.xxx:xxx", "xxx.xxx.xxx.xxx:xxx", "xxx.xxx.xxx.xxx:xxx" ]

  # A config for the cb-network AdminWeb as follows:
  admin_web:
    host: "localhost"
    port: "9999"

  # A config for the cb-network agent as follows:
  cb_network:
    cladnet_id: "xxxx"
    host_id: "" # if host_id is "" (empty string), the cb-network agent will use hostname.

  # A config for the grpc as follows:
  grpc:
    service_endpoint: "localhost:8089"
    server_port: "8089"
    gateway_port: "8088"

  ```

##### log_conf.yaml
- Create `config.yaml` (Use the provided `template-log_conf.yaml`)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/config
  cp template-log_conf.yaml log_conf.yaml
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
cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/admin-web
```

#### Build the admin-web
In the building process, the required packages are automatically installed based on the "go module".
```
go build admin-web.go
```

#### Run the admin-web
```
sudo ./admin-web
```

---

### How to run a cb-network agent based on source code
It was deployed and tested on the "home" directory of Ubuntu 18.04. You can start from YOUR_PROJECT_DIRECTORY.

#### Prepare the config for cb-network controller
##### config.yaml
- Create `config.yaml` (Use the provided `template-config.yaml`)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/config
  cp template-config.yaml config.yaml
  ```
- <ins>**Edit the "xxxx" part**</ins> of `etcd_cluster` and `cb_network` in the text below
- The config.yaml template:
  ```
  # A config for the both cb-network controller and agent as follows:
  etcd_cluster:
    endpoints: [ "xxx.xxx.xxx.xxx:xxx", "xxx.xxx.xxx.xxx:xxx", "xxx.xxx.xxx.xxx:xxx" ]

  # A config for the cb-network AdminWeb as follows:
  admin_web:
    host: "xxx"
    port: "xxx"

  # A config for the cb-network agent as follows:
  cb_network:
    cladnet_id: "xxxx"
    host_id: "xxxx"

  # A config for the grpc as follows:
  grpc:
    service_endpoint: "xxx.xxx.xxx.xxx:xxx"
    server_port: "xxx"
    gateway_port: "xxx"

  ```

##### log_conf.yaml
- Create `config.yaml` (Use the provided `template-log_conf.yaml`)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/config
  cp template-log_conf.yaml log_conf.yaml
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


## Demo: 1st step, to run existing services in multi-cloud

Please refer to the video for more details :-)

NOTE - Please refer to the below for how to run the demo-client used in the video.

[![1st step to run existing services in multi-cloud](https://user-images.githubusercontent.com/7975459/145988454-7e537dcf-b2e2-4560-91ce-eb8455d48772.png)](https://drive.google.com/file/d/1GFuPe-s7IUCbIfLAv-Jkd8JaiQci66nR/view?usp=sharing "Click to watch")

### How to run a demo-client based on source code
It was deployed and tested on the "home" directory of Ubuntu 18.04. You can start from YOUR_PROJECT_DIRECTORY.

#### Prepare the config for the demo-client
##### config.yaml
- Create `config.yaml` (Use the provided `template-config.yaml`)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/test-client/config
  cp template-config.yaml config.yaml
  ```
- <ins>**Edit the "xxxx" part**</ins> of `etcd_cluster` and `grpc` in the text below
- The config.yaml template:
  ```
  # A config for the both cb-network controller and agent as follows:
  etcd_cluster:
    endpoints: [ "xxx.xxx.xxx.xxx:xxx", "xxx.xxx.xxx.xxx:xxx", "xxx.xxx.xxx.xxx:xxx" ]

  # A config for the cb-network AdminWeb as follows:
  admin_web:
    host: "localhost"
    port: "9999"

  # A config for the cb-network agent as follows:
  cb_network:
    cladnet_id: "xxxx"
    host_id: "" # if host_id is "" (empty string), the cb-network agent will use hostname.

  # A config for the grpc as follows:
  grpc:
    service_endpoint: "localhost:8089"
    server_port: "8089"
    gateway_port: "8088"

  ```

#### Change directory
```
cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/test-client
```

#### Build the cladnet service
In the building process, the required packages are automatically installed based on the "go module".
```
go build demo-client.go
```

#### Run the cladnet service
```
sudo ./demo-client
```

