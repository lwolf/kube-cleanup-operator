FROM golang:1.9.0-alpine3.6

RUN apk update && apk add git openssh
RUN go get -u github.com/golang/dep/cmd/dep

COPY . /go/src/github.com/recursionpharma/kube-cleanup-operator
WORKDIR /go/src/github.com/recursionpharma/kube-cleanup-operator
RUN dep ensure
RUN go install -v ./...

ENTRYPOINT ["/go/bin/kube-cleanup-operator"]
