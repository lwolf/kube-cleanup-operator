FROM golang:1.9.0-alpine3.6

RUN apk update && apk add git openssh
RUN go get -u github.com/golang/dep/cmd/dep

COPY . /go/src/github.com/recursionpharma/kube-cleanup-operator
WORKDIR /go/src/github.com/recursionpharma/kube-cleanup-operator
RUN ls
RUN dep ensure
WORKDIR /go/src/github.com/recursionpharma/kube-cleanup-operator/pkg
RUN go install -v ./...
WORKDIR /go/src/github.com/recursionpharma/kube-cleanup-operator/cmd
RUN go install -v ./...
RUN cp /go/bin/cmd /go/bin/kube-cleanup-operator

ENTRYPOINT ["/go/bin/kube-cleanup-operator"]
