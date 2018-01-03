FROM alpine
MAINTAINER Sergey Nuzhdin <ipaq.lw@gmail.com>

RUN addgroup -S kube-operator && adduser -S -g kube-operator kube-operator

USER kube-operator

COPY bin/kube-cleanup-operator .

ENTRYPOINT ["./kube-cleanup-operator"]
