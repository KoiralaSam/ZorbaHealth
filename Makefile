PROTO_DIR := proto
PROTO_SRC := $(shell find $(PROTO_DIR) -name '*.proto')
GO_OUT := .
GO_BIN := $(shell go env GOPATH)/bin
export PATH := $(GO_BIN):$(PATH)
MIGRATIONS_DIR := migrations

.PHONY: generate-proto migrate-up migrate-down sqlc
generate-proto:
	protoc \
		--proto_path=$(PROTO_DIR) \
		--go_out=$(GO_OUT) \
		--go-grpc_out=$(GO_OUT) \
		$(PROTO_SRC)

migrate-up:
	migrate -path $(MIGRATIONS_DIR) -database "$${DATABASE_URL}" up

migrate-down:
	migrate -path $(MIGRATIONS_DIR) -database "$${DATABASE_URL}" down

sqlc:
	sqlc generate
