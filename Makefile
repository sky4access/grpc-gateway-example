SHELL:=/bin/bash
+DESTDIR:=.
VERSION:=1.0.0
BUILD:=$(shell date +%s)
RELEASE:=0
PRODUCT:=greeter
REPO:=github.com/sky4access/grpc-gateway-example

GOARCH:=amd64
GOOS:=darwin

DOCKERNET := layer1

LDFLAGS:=
ifeq ($(GOOS),windows)
BUILDFLAGS:=
else
BUILDFLAGS:=
endif

.PHONY: fetch
fetch: ## get tools
	go get github.com/golang/lint/golint
	go get github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
	go get github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
	go get github.com/golang/protobuf/protoc-gen-go
	go get github.com/mwitkow/go-proto-validators/protoc-gen-govalidators
	go get github.com/fatih/gomodifytags

.PHONY: proto
proto: fetch
	protoc \
		-I./api \
		-I${GOPATH}/src \
		-I${GOPATH}/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
		--go_out=plugins=grpc:./pkg/greeter \
		--grpc-gateway_out=logtostderr=true:./pkg/greeter \
		--swagger_out=logtostderr=true:./assets/swagger \
		--govalidators_out=logtostderr=true:./pkg/greeter \
                ./api/greeter.proto
	# Add `db` tags so that `goqu` can operate on messages directly.
	gomodifytags -w -file pkg/greeter/greeter.pb.go -line 0,999999999 -add-tags db

.PHONY: dep
dep: ## get dependencies and the dep tool if needed
	@if [[ -z `which dep` ]] ; then \
		curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh ; \
	fi
	dep ensure -v

.PHONY: test
test: fetch dep ## run unit tests and code coverage
	golint -set_exit_status $(go list ./...)
	go vet ./...
	go test ./... -v
	go test ./... -cover

.PHONY: build
build: dep ## build go binary
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -v -ldflags \
		"-X main.Version=$(VERSION) -X main.Build=$(BUILD) $(LDFLAGS)" $(BUILDFLAGS) \
		-o $(PRODUCT) "$(REPO)/cmd/$(PRODUCT)"


.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
