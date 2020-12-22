![ubuntu-passing](https://img.shields.io/badge/ubuntu18.04-passing-success)

*Read this in other languages: [English](https://github.com/cloud-barista/cb-larva/blob/master/poc-cb-net/README.md), [한국어](https://github.com/cloud-barista/cb-larva/blob/master/poc-cb-net/README.KR.md)*

# cb-network

cb-network는 연구 개발 중이고, **cb-subnet**에 대한 개념 증명을 진행하고 있습니다.

## cb-network 개요
cb-network는 클라우드바리스타의 글로벌 네트워크 서비스 입니다. 목표는 <ins>**멀티 CSP의 이종 네트워크 상에서 cb-network가 통일되고 효율적인 글로벌 네트워크 서비스 제공**</ins>하는 것 입니다. CSP의 네트워크로부터 독립적인 네트워크 서비스를 만드는 것이지요. 

cb-network는 cb-subnet, cb-dns, and cb-loadbalancer을 포함하고 있는데요. 추가 기능/아이템도 언제든지 환영합니다.

<p align="center">
  <img src="https://user-images.githubusercontent.com/7975459/99206719-7ea7c500-27ff-11eb-96f3-bc912bf7143a.png">
</p>

저희는 현재 cb-network의 cb-subnet을 주로 연구 개발 중이고, cb-dns와 cb-loadbalancer 연구 개발은 예정되어 있습니다.  
**cb-subnet**은 다중 VM을 위해 공통의 네트워크 생성 작업을 수행하고, VM간 통신을 지원합니다.

## cb-network Server 시작하기
### 필수 사항(Prerequisites)
#### Golang 1.15.3 설치
참고, [Go Setup Script](https://github.com/cb-contributhon/cb-coffeehouse/tree/master/scripts/go-setup)
```
wget https://raw.githubusercontent.com/cb-contributhon/cb-coffeehouse/master/scripts/go-setup/go1.15.3-setup.sh
source go1.15.3-setup.sh
```

### cb-network Server 실행 방법
아래 과정은 Ubuntu 18.04의 "home" 디렉토리에서 진행하였습니다. 별도의 프로젝트 디렉토리에서 진행하는 것도 가능합니다.

#### CB-Larva 저장소 클론하기
```
git clone https://github.com/cloud-barista/cb-larva.git
```

#### MQTT Broker config 만들기
##### "configs" 디렉토리 생성
```
mkdir $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/configs
```
##### MQTT Broker에 대한 config 파일 생성
- 파일명: "mqtt-broker.json"
- 내용:
```
{
  "MQTTBrokerIP": "xxxx",
  "MQTTBrokerPort": "xxxx"
}
```
**<ins>"xxxx" 부분은 변경 하시기 바랍니다.</ins>**

#### 디렉토리 경로 변경
```
cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/server
```

#### cb-network Server 빌드
빌드 과정에서 필요한 패키지를 자동으로 설치합니다. (go module이 참 편리하네요 ㅎㅎ)
```
go build server.go
```

#### cb-network Server 실행
```
sudo ./server
```


## cb-network Agent 
### 필수 사항(Prerequisites)
#### Golang 1.15.3 설치
**만약, 위 cb-network Server 부분에서 Golang 1.15.3을 설치했다면, 이 과정을 건너뛸 수 있습니다.**
Please refer to [Go Setup Script](https://github.com/cb-contributhon/cb-coffeehouse/tree/master/scripts/go-setup)
```
wget https://raw.githubusercontent.com/cb-contributhon/cb-coffeehouse/master/scripts/go-setup/go1.15.3-setup.sh
source go1.15.3-setup.sh
```

### cb-network Agent 실행 방법
아래 과정은 Ubuntu 18.04의 "home" 디렉토리에서 진행하였습니다. 별도의 프로젝트 디렉토리에서 진행하는 것도 가능합니다.

#### CB-Larva 저장소 클론하기
```
git clone https://github.com/cloud-barista/cb-larva.git
```

#### MQTT Broker config 만들기
**만약, 위 cb-network Server 부분에서 "configs" 디렉토리 및 "mqtt-broker.json"파일을 만드셨다면, 이 과정을 건너뛸 수 있습니다.**

##### "configs" 디렉토리 생성
```
mkdir $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/configs
```
##### MQTT Broker에 대한 config 파일 생성
- 파일명: "mqtt-broker.json"
- 내용:
```
{
  "MQTTBrokerIP": "xxxx",
  "MQTTBrokerPort": "xxxx"
}
```
**<ins>"xxxx" 부분은 변경 하시기 바랍니다.</ins>**

#### 디렉토리 경로 변경
```
cd $YOUR_PROJECT_DIRECTORY/cb-larva/poc-cb-net/cmd/agent
```

#### cb-network Agent 빌드
빌드 과정에서 필요한 패키지를 자동으로 설치합니다. (go module이 참 편리하네요 ㅎㅎ)

```
go build agent.go
```

#### cb-network Agent 
```
sudo ./agent
```
