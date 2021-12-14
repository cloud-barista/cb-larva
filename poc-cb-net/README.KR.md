![ubuntu-passing](https://img.shields.io/badge/ubuntu18.04-passing-success)

*Read this in other languages: [English](https://github.com/cloud-barista/cb-larva/blob/master/poc-cb-net/README.md), [한국어](https://github.com/cloud-barista/cb-larva/blob/master/poc-cb-net/README.KR.md)*

**[바로가기]**
- [Cloud-Barista Network 개요](#cloud-barista-network-개요)
- [Cloud Adaptive Network 소개](#cloud-adaptive-network-소개)
- [cb-network 시스템 시작하기](#cb-network-시스템-시작하기)
  - [필수 사항(Prerequisites)](#필수-사항prerequisites)
  - [소스 코드 기반 cb-network controller 구동](#소스-코드-기반-cb-network-controller-구동)
  - [소스 코드 기반 cladnet service 구동](#소스-코드-기반-cladnet-service-구동)
  - [소스 코드 기반 admin-web 구동](#소스-코드-기반-admin-web-구동)
  - [소스 코드 기반 cb-network agent 구동](#소스-코드-기반-cb-network-agent-구동)
- [데모: 멀티클라우드에 기존 서비스를 올리기 위한 첫 걸음](#데모-멀티클라우드에-기존-서비스를-올리기-위한-첫-걸음)



## Cloud-Barista Network 개요

연구 개발 중인 Cloud-Barista Network (cb-network)는 
<ins>**전세계 클라우드 인프라를 엮기 위해 클라우드 네트워크(e.g., VPC, vNet)의 상이함과 변동성을 완화한 글로벌 스케일 네트워크**</ins> 입니다. 

앞으로 네트워크 관련 기술(e.g., Subnet, DNS, and Load balancer)을 추가하며 점차 확장해 나갈 가장 상위/넓은 개념 입니다.
CSP의 네트워크로부터 독립적인 클라우드바리스타를 위한 가상 네트워크라고 말씀 드릴 수 있을 것 같네요.

이후 cb-network는 주로 시스템 또는 비전을 나타내고, Cloud Adaptive Network (CLADNet) 연구 개발 중인 기술을 나타낼 것입니다.

<p align="center">
  <img src="https://user-images.githubusercontent.com/7975459/145977420-1e5af8b1-bf87-4282-917c-9c982915c332.png">
</p>


## Cloud Adaptive Network 소개

간단히 말하면, Cloud Adaptive Network (CLADNet)는 멀티클라우드의 <ins>**다양한 네트워크에 적응가능한**</ins> 오버레이 네트워크 입니다.
 
논리적인 노드 그룹에 동일 네트워크(e.g., Subnet) 및 관련 핵심 기능을 제공합니다. 
쉽게 말해, **CLADNet (cb-cladnet)은** 다중 VM을 위한 공통의 네트워크를 제공하고, VM간 통신을 지원합니다.

<p align="center">
  <img src="https://user-images.githubusercontent.com/7975459/122491196-8130fe00-d01e-11eb-881e-1d3d3a2aa0c4.png">
</p>

### CLADNet의 지향점
- **Adaptive**: 여러 사업자의 상이한 Cloud Network에 적응 가능한 네트워크
- **Fault tolerant**: 사업자 및 리전 이슈에 대비하는 글로벌 장애 허용 네트워크 
- **Lightweight**: 호스트(VM) 자원 사용을 최소화하는 경량한 네트워크
- **Handy**: 사용자가 쉽고 빠르게 사용할 수 있는 네트워크

### CLADNet의 구조

- Event-driven 아키텍처: 분산 Key-Value store 기반의 Event-driven 아키텍처로 데이터의 변경, 생성, 삭제(CUD)시 발생하는 
                        서비스의 의미있는 변화를 바탕으로 효율적인 워크플로우 수행
  - Microservice Architecture를 향해 나아가는 중

- 메쉬(Mesh)형 토폴로지: 오버레이 네트워크를 메쉬형 토폴로지로 구성하여 중개 노드의 위치에 따른 성능 차이를 최소화함
  - 통신 성능을 높히기 위해 연구 중


## cb-network 시스템 시작하기
cb-network 시스템을 시작하기 위해 필요한 준비사항 및 각 컴포넌트 실행 방법에 대해 설명한다.
`cb-network controller`, `cb-network cladnet-service`, `cb-network admin-web`, 및 `distributed key-value store`는 동일 노드에서 구동 가능하고,
`cb-network agent`는 각각 서로 다른 호스트(VM)에서 구동해야 한다.
### 필수 사항(Prerequisites)
#### 패키지/도구 설치
```
sudo apt update -y
sudo apt dist-upgrade -y
sudo apt install git -y
```

#### Golang 설치
참고: [Go Setup Script](https://github.com/cloud-barista/cb-coffeehouse/tree/master/scripts/golang)
```
wget https://raw.githubusercontent.com/cloud-barista/cb-coffeehouse/master/scripts/golang/go-installation.sh
source go-installation.sh
```

#### CB-Larva 저장소 클론
```
git clone https://github.com/cloud-barista/cb-larva.git
```

#### Distributed key-value store 배치
cb-network 시스템은 분산 키-값 저장소를 필요로 합니다. 
여기서는 `etcd`를 활용하였고, 테스트를 위해 단일-노드 클러스터를 배치했습니다.

아래 링크 참고:
- [etcd 3.5 - Run etcd clusters inside containers](https://etcd.io/docs/v3.5/op-guide/container/)
- [etcd 3.5 - Quickstart](https://etcd.io/docs/v3.5/quickstart/)
- [etcd 3.5 - Demo](https://etcd.io/docs/v3.5/demo/)

---

### 소스 코드 기반 cb-network controller 구동
아래 과정은 Ubuntu 18.04의 "home" 디렉토리를 기준으로 진행 하였습니다.

#### cb-network controller 관련 설정파일 준비
##### config.yaml
- config.yaml 생성(제공된 `template-config.yaml`을 활용)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/config
  cp template-config.yaml config.yaml
  ```
- 아래 내용에서 `etcd_cluster`의 **<ins>"xxxx" 부분 수정</ins>**
- 내용:
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

  demo_app:
    is_run: false
  ```

##### log_conf.yaml
- log_conf.yaml 생성(제공된 `template-log_conf.yaml`을 활용)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/config
  cp template-log_conf.yaml log_conf.yaml
  ```
- 필요시 아래 내용에서 `cblog` > `loglevel` 수정
- 내용:
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
#### 디렉토리 경로 변경
```
cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/controller
```

#### cb-network controller 빌드
빌드 과정에서 필요한 패키지를 자동으로 설치합니다. (go module이 참 편리하네요 ㅎㅎ)
```
go build controller.go
```

#### cb-network controller 실행
```
sudo ./controller
```

---

### 소스 코드 기반 cladnet service 구동
아래 과정은 Ubuntu 18.04의 "home" 디렉토리를 기준으로 진행 하였습니다.

#### cladnet service 관련 설정파일 준비
##### config.yaml
- config.yaml 생성(제공된 `template-config.yaml`을 활용)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/config
  cp template-config.yaml config.yaml
  ```
- 아래 내용에서 `etcd_cluster` 및 `grpc` 의 **<ins>"xxxx" 부분 수정</ins>**
- 내용:
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

  demo_app:
    is_run: false
  ```

##### log_conf.yaml
- log_conf.yaml 생성(제공된 `template-log_conf.yaml`을 활용)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/config
  cp template-log_conf.yaml log_conf.yaml
  ```
- 필요시 아래 내용에서 `cblog` > `loglevel` 수정
- 내용:
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
#### 디렉토리 경로 변경
```
cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/service
```

#### cladnet service 빌드
빌드 과정에서 필요한 패키지를 자동으로 설치합니다.
```
go build cladnet-service.go
```

#### cladnet service 실행
```
sudo ./cladnet-service
```

---

### 소스 코드 기반 admin-web 구동
아래 과정은 Ubuntu 18.04의 "home" 디렉토리를 기준으로 진행 하였습니다.

#### admin-web 관련 설정파일 준비
##### config.yaml
- config.yaml 생성(제공된 `template-config.yaml`을 활용)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/config
  cp template-config.yaml config.yaml
  ```
- 아래 내용에서 `etcd_cluster`, `admin_web` 및 `grpc` 의 **<ins>"xxxx" 부분 수정</ins>**
- 내용:
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

  demo_app:
    is_run: false
  ```

##### log_conf.yaml
- log_conf.yaml 생성(제공된 `template-log_conf.yaml`을 활용)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/config
  cp template-log_conf.yaml log_conf.yaml
  ```
- 필요시 아래 내용에서 `cblog` > `loglevel` 수정
- 내용:
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
#### 디렉토리 경로 변경
```
cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/admin-web
```

#### admin-web 빌드
빌드 과정에서 필요한 패키지를 자동으로 설치합니다.
```
go build admin-web.go
```

#### admin-web 실행
```
sudo ./admin-web
```

---

### 소스 코드 기반 cb-network agent 구동
아래 과정은 Ubuntu 18.04의 "home" 디렉토리를 기준으로 진행 하였습니다.

#### cb-network agent 관련 설정파일 준비
##### config.yaml
- config.yaml 생성(제공된 `template-config.yaml`을 활용)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/config
  cp template-config.yaml config.yaml
  ```
- 아래 내용에서 `etcd_cluster` 및 `cb_network`의 **<ins>"xxxx" 부분 수정</ins>**
  - 주의!!! agent마다 `cb_network` > `host_id`를 다르게 
- 내용:
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

  demo_app:
    is_run: false
  ```

##### log_conf.yaml
- log_conf.yaml 생성(제공된 `template-log_conf.yaml`을 활용)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/config
  cp template-log_conf.yaml log_conf.yaml
  ```
- 필요시 아래 내용에서 `cblog` > `loglevel` 수정
- 내용:
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

#### 디렉토리 경로 변경
```
cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/agent
```

#### cb-network agent 빌드
빌드 과정에서 필요한 패키지를 자동으로 설치합니다.

```
go build agent.go
```

#### cb-network agent 
```
sudo ./agent
```


## 데모: 멀티클라우드에 기존 서비스를 올리기 위한 첫 걸음

자세한 내용은 영상을 참고해 주세요 :-)

비고 - 영상에서 사용한 demo-client를 구동하는 방법은 아래를 참고해 주세요.

[![멀티클라우드에 기존 서비스를 올리기 위한 첫 걸음](https://user-images.githubusercontent.com/7975459/145988454-7e537dcf-b2e2-4560-91ce-eb8455d48772.png)](https://drive.google.com/file/d/1GFuPe-s7IUCbIfLAv-Jkd8JaiQci66nR/view?usp=sharing "Click to watch")

### 소스 코드 기반 demo-client 구동
아래 과정은 Ubuntu 18.04의 "home" 디렉토리를 기준으로 진행 하였습니다.

#### demo-client 관련 설정파일 준비
##### config.yaml
- config.yaml 생성(제공된 `template-config.yaml`을 활용)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/test-client/config
  cp template-config.yaml config.yaml
  ```
- 아래 내용에서 `etcd_cluster` 및 `grpc` 의 **<ins>"xxxx" 부분 수정</ins>**
- 내용:
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

  demo_app:
    is_run: false
  ```

#### 디렉토리 경로 변경
```
cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/test-client
```

#### demo-client 빌드
빌드 과정에서 필요한 패키지를 자동으로 설치합니다.
```
go build demo-client.go
```

#### demo-client 실행
```
sudo ./demo-client
```
