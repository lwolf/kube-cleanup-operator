NAME := kube-cleanup-operator
AUTHOR=lwolf
VERSION ?= 0.6.0
REGISTRY ?= quay.io
GIT_SHA=$(shell git --no-pager describe --always --dirty)
COMMIT_TIME=$(shell git show --format=%ct --no-patch)
LFLAGS ?= -X main.gitsha=${GIT_SHA} -X main.committed=${COMMIT_TIME}
ROOT_DIR=${PWD}
GOVERSION ?= 1.14.0
HARDWARE=$(shell uname -m)

.PHONY: build docker static release install_deps

default: build

golang:
	@echo "--> Go Version"
	@go version

install_deps:
	go mod tidy
	go mod vendor

build: golang
	@echo "--> Compiling the project"
	@mkdir -p bin
	go build -mod=vendor -ldflags "${LFLAGS}" -o bin/$(NAME) ./cmd

static: golang 
	@echo "--> Compiling the static binary"
	@mkdir -p bin
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -mod=vendor -a -tags netgo -ldflags "-w ${LFLAGS}" -o bin/${NAME} ./cmd

docker-build:
	@echo "--> Compiling the project"
	docker run --rm \
		-v ${ROOT_DIR}:/go/src/github.com/${AUTHOR}/${NAME} \
		-w /go/src/github.com/${AUTHOR}/${NAME} \
		-e GOOS=linux golang:${GOVERSION} \
		make static

docker-release:
	@echo "--> Building a release image"
	@$(MAKE) static
	@$(MAKE) docker
	@docker push ${REGISTRY}/${AUTHOR}/${NAME}:${VERSION}

docker:
	@echo "--> Building the docker image"
	docker build -t ${REGISTRY}/${AUTHOR}/${NAME}:${VERSION} .

release: static
	mkdir -p release
	gzip -c bin/${NAME} > release/${NAME}_${VERSION}_linux_${HARDWARE}.gz
	rm -f release/${NAME}

clean:
	rm -rf ./bin 2>/dev/null
	rm -rf ./release 2>/dev/null

format:
	@echo "--> Running go fmt"
	@gofmt -s -w ./
