default: stub reverse_proxy

all: stub reverse_proxy docs_protocol docs_swagger

apis: stub reverse_proxy

docs: docs_protocol docs_swagger

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

docs_protocol:
	protoc -I ./proto \
	--doc_out=./gen/docs/cbnetwork --doc_opt=markdown,cloud-adaptive-network-service.md \
	./proto/cbnetwork/cloud_adaptive_network.proto
	cp ./gen/docs/cbnetwork/cloud-adaptive-network-service.md ../../docs/

docs_swagger:
	protoc -I ./proto \
	--openapiv2_out ./gen/docs \
	--openapiv2_opt logtostderr=true \
	./proto/cbnetwork/cloud_adaptive_network.proto
	cp ./gen/docs/cbnetwork/cloud_adaptive_network.swagger.json ../../docs/
