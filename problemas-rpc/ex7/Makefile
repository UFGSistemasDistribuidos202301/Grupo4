all:
	protoc --go_out=./server/pb --go_opt=paths=source_relative \
		--go-grpc_out=./server/pb --go-grpc_opt=paths=source_relative \
		./proto/rpc.proto
	python -m grpc_tools.protoc \
		-I. \
		--python_out=./client \
		--pyi_out=./client \
		--grpc_python_out=./client \
		./proto/rpc.proto

.PHONY: all
