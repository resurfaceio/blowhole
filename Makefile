generate_grpc_code:
	protoc \
    --go_out=DistributedServices \
    --go_opt=paths=source_relative \
    --go-grpc_out=DistributedServices \
    --go-grpc_opt=paths=source_relative \
    blowhole.proto