default:
	go build -mod=mod -o ./cmd/admin-web/admin-web ./cmd/admin-web/admin-web.go
	go build -mod=mod -o ./cmd/agent/agent ./cmd/agent/agent.go
	go build -mod=mod -o ./cmd/controller/controller ./cmd/controller/controller.go
	go build -mod=mod -o ./cmd/service/cladnet-service ./cmd/service/cladnet-service.go
	go build -mod=mod -o ./cmd/test-client/test-client ./cmd/test-client/test-client.go
# cc:
#  	GOOS=linux GOARCH=arm go build -mod=mod -o cb-tumblebug-arm
# swag swagger:
# 	~/go/bin/swag i -o ./api/rest/docs
# proto protobuf pb:
# 	cd api/grpc/protobuf && $(MAKE) regenerate
# cbadm:
# 	cd api/grpc/cbadm && $(MAKE)
# run:
# 	./cb-tumblebug
# clean:
# 	rm -v cb-tumblebug cb-tumblebug-arm