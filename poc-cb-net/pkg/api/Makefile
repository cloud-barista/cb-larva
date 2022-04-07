default: stub reverse_proxy

all: stub reverse_proxy doc

stub:
	protoc -I ./proto \
	--go_out ./gen/go --go_opt paths=source_relative \
	--go-grpc_out ./gen/go --go-grpc_opt paths=source_relative \
	./proto/cbnetwork/cloud_adaptive_network.proto

reverse_proxy:
	protoc -I ./proto \
	--go_out ./gen/go --go_opt paths=source_relative \
	--go-grpc_out ./gen/go --go-grpc_opt paths=source_relative \
	--grpc-gateway_out ./gen/go --grpc-gateway_opt paths=source_relative \
	./proto/cbnetwork/cloud_adaptive_network.proto

doc:
	protoc -I ./proto \
	--doc_out=./gen/doc --doc_opt=markdown,cloud-adaptive-network-service.md \
	./proto/cbnetwork/cloud_adaptive_network.proto

openapiv2:
	protoc -I ./proto \
	--openapiv2_out ./gen/openapiv2 \
	--openapiv2_opt logtostderr=true \
	./proto/cbnetwork/cloud_adaptive_network.proto