all:
	go build .

proto:
	protoc --go_out=./pb --go_opt=paths=source_relative \
		--go-grpc_out=./pb --go-grpc_opt=paths=source_relative \
		./rpc.proto

.PHONY: all proto
