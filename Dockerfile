FROM golang:1.14.4-alpine3.12 AS build

RUN apk update && \
    apk add build-base git

COPY . /build
WORKDIR /build

RUN make install_deps
RUN make build

FROM alpine
MAINTAINER Sergey Nuzhdin <ipaq.lw@gmail.com>

RUN addgroup -S kube-operator && adduser -S -g kube-operator kube-operator

USER kube-operator

COPY --from=build /build/bin/kube-cleanup-operator .

ENTRYPOINT ["./kube-cleanup-operator"]
