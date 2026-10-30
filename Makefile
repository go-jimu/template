BUF_VERSION ?= v1.67.0
PROTOC_GEN_GO_VERSION ?= v1.36.11
PROTOC_GEN_CONNECT_GO_VERSION ?= v1.20.0

.PHONY: test
test:
	go test -race -covermode=atomic -coverprofile=coverage.txt ./...

# integration starts MySQL with testcontainers, so it needs Docker.
.PHONY: integration
integration:
	go test -race -tags integration -run MySQL ./internal/business/user/infrastructure/...

.PHONY: vet
vet:
	go vet ./...

.PHONY: generate
generate:
	buf lint
	buf generate

.PHONY: verify
verify: generate test vet

.PHONY: server
server:
	go run ./cmd/server

.PHONY: client
client:
	go run ./cmd/client

.PHONY: mod-tidy
mod-tidy:
	go mod tidy

.PHONY: tools
tools:
	go install github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION)
	go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO_VERSION)
	go install connectrpc.com/connect/cmd/protoc-gen-connect-go@$(PROTOC_GEN_CONNECT_GO_VERSION)
