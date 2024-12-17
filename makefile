build_proto:
	protoc --go_out=./DSM_protos --go_opt=paths=source_relative \
    --go-grpc_out=./DSM_protos --go-grpc_opt=paths=source_relative \
	DSM.proto

