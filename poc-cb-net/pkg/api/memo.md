## Issues of compiling protobuf and sort of solution for it

Issues:
- There is little relevant information.
- The information is different.
- Commands for compile protobuf are various or different.
- The required Go packages are updated and changed.

비고: protobuf를 buf로 compile하는 경우 - yaml 설정을 통해 googleapis를 알아서 포함하도록 할 수 있음
비고: protobuf를 protoc로 compile하는 경우 - 직접 디렉토리 구조를 설정하고, googleapis repository에서 필요한 proto 파일을 내려받아 복사해야함

### Method 1: the separated files
#### Required Go packages
```
go get github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway
go get google.golang.org/protobuf/cmd/protoc-gen-go
go get google.golang.org/grpc/cmd/protoc-gen-go-grpc
```

#### Stub generation
```
protoc -I ./proto \
--go_out ./gen/go --go_opt paths=source_relative \
--go-grpc_out ./gen/go --go-grpc_opt paths=source_relative \
./proto/cbnetwork/cloud_adaptive_network.proto
```

Generated files:
- cloud_adaptive_network_grpc.pb.go
- cloud_adaptive_network.pb.go

#### Stub and reverse proxy generation
```
protoc -I ./proto \
--go_out ./gen/go --go_opt paths=source_relative \
--go-grpc_out ./gen/go --go-grpc_opt paths=source_relative \
--grpc-gateway_out ./gen/go --grpc-gateway_opt paths=source_relative \
./proto/cbnetwork/cloud_adaptive_network.proto
```

Generated files:
- cloud_adaptive_network_grpc.pb.go
- cloud_adaptive_network.pb.go
- cloud_adaptive_network.pb.gw.go


### Method 2: the integrated files

`xxxx_grpc.pb.go`와 `xxxx.pb.go`가 한 파일로 통합된 형태

#### Required Go packages
비고: 위에서 패키지를 설치한 후 진행하여 요구되는 패키지가 다를 수 있음

```
go get -u github.com/golang/protobuf/protoc-gen-go
```

#### Stub generation
```
protoc -I ./proto \
--go_out=plugins=grpc:./gen/go --go_opt=paths=source_relative \
./proto/cbnetwork/cloud_adaptive_network.proto
```

Generated files:
- cloud_adaptive_network.pb.go 


#### Stub and reverse proxy generation
```
protoc -I ./proto \
--go_out=plugins=grpc:./gen/go --go_opt paths=source_relative \
--grpc-gateway_out ./gen/go --grpc-gateway_opt paths=source_relative \
./proto/cbnetwork/cloud_adaptive_network.proto
```

Generated files:
- cloud_adaptive_network.pb.go
- cloud_adaptive_network.pb.gw.go

References: 
- [gRPC-Gateway](https://grpc-ecosystem.github.io/grpc-gateway/)
- gRPC 시작에서 운영까지 OREILLY (publised in 20210104)
- [gRPC-Gateway, GitHub](https://github.com/grpc-ecosystem/grpc-gateway)
- [googleapis](https://github.com/googleapis/googleapis/tree/master/google/api), especially `annotations.proto`, `field_behavior.proto`, `http.proto`, and `httpbody.proto`
- [Go gRPC 서버에 REST API 요청 주고 받기 [grpc-gateway]](https://codeac.tistory.com/130)
- [gRPC 소개 및 go 예제](https://lejewk.github.io/grpc-go-example/)
- [Go 언어에서 GRPC 프로그래밍 하기 #1 - 기초 편](https://alnova2.tistory.com/1373)