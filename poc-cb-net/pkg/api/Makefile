regenerate:
	protoc \
	--proto_path=./protobuf \
	protobuf/cloud_adaptive_network.proto \
	--go_opt=paths=source_relative \
	--go_out=plugins=grpc:./gen/go
