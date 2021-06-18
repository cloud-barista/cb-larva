![ubuntu-passing](https://img.shields.io/badge/ubuntu18.04-passing-success)

*Read this in other languages: [English](https://github.com/cloud-barista/cb-larva/blob/master/poc-cb-net/README.md), [한국어](https://github.com/cloud-barista/cb-larva/blob/master/poc-cb-net/README.KR.md)*

# Cloud-Barista Network

연구 개발 중인 Cloud-Barista Network (cb-network)는 
<ins>**전세계 클라우드 인프라를 엮기 위해 클라우드 네트워크(e.g., VPC, vNet)의 상이함과 변동성을 완화한 글로벌 스케일 네트워크**</ins> 입니다. 

앞으로 네트워크 관련 기술(e.g., Subnet, DNS, and Load balancer)을 추가하며 점차 확장해 나갈 가장 상위/넓은 개념 입니다.
CSP의 네트워크로부터 독립적인 클라우드바리스타를 위한 가상 네트워크라고 말씀 드릴 수 있을 것 같네요.

이와 같은, cb-network라는 큰 개념 아래 Cloud Adaptive Network (CLADNet)를 연구 개발 중 입니다.

<p align="center">
  <img src="https://user-images.githubusercontent.com/7975459/122491196-8130fe00-d01e-11eb-881e-1d3d3a2aa0c4.png">
</p>


## Cloud Adaptive Network

Cloud Adaptive Network는 멀티클라우드의 <ins>**다양한 네트워크에 적응가능한**</ins> 오버레이 네트워크 입니다.
 
논리적인 노드 그룹에 동일 네트워크(e.g., Subnet) 및 관련 핵심 기능을 제공합니다. 
쉽게 말해, **CLADNet (cb-cladnet)은** 다중 VM을 위한 공통의 네트워크를 제공하고, VM간 통신을 지원합니다.

### CLADNet의 지향점
- Adaptive: 여러 사업자의 상이한 Cloud Network에 적응 가능한 네트워크
- Fault tolerant: 사업자 및 리전 이슈에 대비하는 글로벌 장애 허용 네트워크 
- Lightweight: 호스트(VM) 자원 사용을 최소화하는 경량한 네트워크
- Handy: 사용자가 쉽고 빠르게 사용할 수 있는 네트워크

### CLADNet의 구조

- Event-driven 아키텍처: 분산 Key-Value store 기반의 Event-driven 아키텍처로 데이터의 변경, 생성, 삭제(CUD)시 발생하는 
                        서비스의 의미있는 변화를 바탕으로 효율적인 워크플로우 수행
  - Microservice Architecture를 향해 나아가는 중

- 메쉬(Mesh)형 토폴로지: 오버레이 네트워크를 메쉬형 토폴로지로 구성하여 중개 노드의 위치에 따른 성능 차이를 최소화함
  - IPSec 등 타 프로토콜 적용을 위해 Pluggable Interface로 구조 개선 예정


## cb-network 시작하기
### 필수 사항(Prerequisites)
#### 패키지/도구 설치
- `sudo apt update -y`
- `sudo apt dist-upgrade -y`
- `sudo apt install git -y`

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


### 소스 코드 기반 cb-network server 구동
아래 과정은 Ubuntu 18.04의 "home" 디렉토리를 기준으로 진행 하였습니다.

#### cb-network server 관련 설정파일 준비
##### config.yaml
- config.yaml 생성(제공된 `template)config.yaml`을 활용)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/configs
  cp template)config.yaml config.yaml
  ```
- 아래 내용에서 `etcd_cluster` 및 `admin_web`의 **<ins>"xxxx" 부분 수정</ins>**
- 내용:
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
- log_conf.yaml 생성(제공된 `template)log_conf.yaml`을 활용)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/configs
  cp template)log_conf.yaml log_conf.yaml
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
cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/server
```

#### cb-network server 빌드
빌드 과정에서 필요한 패키지를 자동으로 설치합니다. (go module이 참 편리하네요 ㅎㅎ)
```
go build server.go
```

#### cb-network server 실행
```
sudo ./server
```


### 소스 코드 기반 cb-network agent 구동
아래 과정은 Ubuntu 18.04의 "home" 디렉토리를 기준으로 진행 하였습니다.

#### cb-network agent 관련 설정파일 준비
##### config.yaml
- config.yaml 생성(제공된 `template)config.yaml`을 활용)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/configs
  cp template)config.yaml config.yaml
  ```
- 아래 내용에서 `etcd_cluster` 및 `cb_network`의 **<ins>"xxxx" 부분 수정</ins>**
  - agent마다 `cb_network` > `host_id`를 다르게 
- 내용:
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
- log_conf.yaml 생성(제공된 `template)log_conf.yaml`을 활용)
  ```
  cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/configs
  cp template)log_conf.yaml log_conf.yaml
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
