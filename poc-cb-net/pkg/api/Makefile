default: stub reverse_proxy

all: stub reverse_proxy docs_protocol docs_swagger

apis: stub reverse_proxy

docs: docs_protocol docs_swagger

stub:
	protoc -I ./proto \
	--go_out ./gen/go --go_opt paths=source_relative \
	--go-grpc_out ./gen/go --go-grpc_opt paths=source_relative \
	./proto/cloud_barista_network.proto

reverse_proxy:
	protoc -I ./proto \
	--go_out ./gen/go --go_opt paths=source_relative \
	--go-grpc_out ./gen/go --go-grpc_opt paths=source_relative \
	--grpc-gateway_out ./gen/go --grpc-gateway_opt paths=source_relative \
	./proto/cloud_barista_network.proto

docs_protocol:
	protoc -I ./proto \
	--doc_out=./gen/docs --doc_opt=markdown,cloud-barista-network-service.md \
	./proto/cloud_barista_network.proto
	cp ./gen/docs/cloud-barista-network-service.md ../../docs/

docs_swagger:
	protoc -I ./proto \
	--openapiv2_out ./gen/docs \
	--openapiv2_opt logtostderr=true \
	./proto/cloud_barista_network.proto
	cp ./gen/docs/cloud_barista_network.swagger.json ../../docs/
