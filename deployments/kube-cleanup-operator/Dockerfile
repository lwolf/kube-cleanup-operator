FROM gcr.io/outreach-docker/alpine/golang:latest as builder

RUN apk update && \
    apk add build-base git

COPY . /build
WORKDIR /build

RUN make install_deps
RUN make build

FROM gcr.io/outreach-docker/alpine/golang:latest as builder
MAINTAINER Sergey Nuzhdin <ipaq.lw@gmail.com>

RUN addgroup -S kube-operator && adduser -S -g kube-operator kube-operator

USER kube-operator

COPY --from=build /build/bin/kube-cleanup-operator .

ENTRYPOINT ["./kube-cleanup-operator"]
