default: controller service admin_web agent demo_client perf_eval_client

controller:
	go build -mod=mod -o ./cmd/controller/controller ./cmd/controller/controller.go

service:
	go build -mod=mod -o ./cmd/service/service ./cmd/service/service.go

admin_web:
	go build -mod=mod -o ./cmd/admin-web/admin-web ./cmd/admin-web/admin-web.go

agent:
	go build -mod=mod -o ./cmd/agent/agent ./cmd/agent/agent.go

demo_client:
	go build -mod=mod -o ./cmd/test-client/demo-client/demo-client ./cmd/test-client/demo-client/demo-client.go

perf_eval_client:
	go build -mod=mod -o ./cmd/test-client/perf-eval-client/perf-eval-client ./cmd/test-client/perf-eval-client/perf-eval-client.go

production: 
	@echo "Build for production"
# Note - Using cgo write normal Go code that imports a pseudo-package "C". I may not need on cross-compiling.
# Note - You can find possible platforms by 'go tool dist list' for GOOS and GOARCH
# Note - Using the -ldflags parameter can help set variable values at compile time.
# Note - Using the -s and -w linker flags can strip the debugging information.
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s -w' -o ./cmd/controller/controller ./cmd/controller/controller.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s -w' -o ./cmd/service/service ./cmd/service/service.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s -w' -o ./cmd/admin-web/admin-web ./cmd/admin-web/admin-web.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s -w' -o ./cmd/agent/agent ./cmd/agent/agent.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s -w' -o ./cmd/test-client/demo-client/demo-client ./cmd/test-client/demo-client/demo-client.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags '-s -w' -o ./cmd/test-client/demo-client/perf-eval-client ./cmd/test-client/perf-eval-client.go

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