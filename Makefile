OPERATOR_NAME  := kube-cleanup-operator
VERSION := $(shell date +%Y%m%d%H%M)
IMAGE := lwolf/$(OPERATOR_NAME)

.PHONY: install_deps build build-image

install_deps:
	dep ensure

build:
	rm -rf bin/%/$(OPERATOR_NAME)
	go build -v -i -o bin/$(OPERATOR_NAME) ./cmd

bin/%/$(OPERATOR_NAME):
	rm -rf bin/%/$(OPERATOR_NAME)
	GOOS=$* GOARCH=amd64 go build -v -i -o bin/$*/$(OPERATOR_NAME) ./cmd

build-image: bin/linux/$(OPERATOR_NAME)
	docker build . -t $(IMAGE):$(VERSION)
