.PHONY: proto build test
MODULE := github.com/tollofsen/bazooka

# Regenerate Go and Python stubs. Requires protoc-gen-go, protoc-gen-go-grpc
# on PATH and `pip install grpcio-tools`.
proto:
	python -m grpc_tools.protoc -Iproto \
		--go_out=. --go_opt=module=$(MODULE) \
		--go-grpc_out=. --go-grpc_opt=module=$(MODULE) \
		--python_out=automation --grpc_python_out=automation --pyi_out=automation \
		proto/bazooka/v1/bazooka.proto

build:
	go build -o bin/ ./backend/cmd/bazooka-server ./frontend/tui

test:
	go test ./...
