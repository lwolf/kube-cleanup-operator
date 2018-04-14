# Kubernetes cleanup operator

[![Build Status](https://travis-ci.org/lwolf/kube-cleanup-operator.svg?branch=master)](https://travis-ci.org/lwolf/kube-cleanup-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/lwolf/kube-cleanup-operator)](https://goreportcard.com/report/github.com/lwolf/kube-cleanup-operator)
[![Docker Repository on Quay](https://quay.io/repository/lwolf/kube-cleanup-operator/status "Docker Repository on Quay")](https://quay.io/repository/lwolf/kube-cleanup-operator)

Experimental Kubernetes Operator to automatically delete completed Jobs and their Pods.
Controller listens for changes in Pods created by Jobs and deletes it on Completion.

Some defaults:
* All Namespaces are monitored by default
* Only Pods created by Jobs are monitored

## Usage

![screensharing](http://g.recordit.co/aDU52FJIwP.gif)

```
# remember to change namespace in RBAC manifests for monitoring namespaces other than "default"

kubectl create -f https://raw.githubusercontent.com/lwolf/kube-cleanup-operator/master/deploy/rbac.yaml

# create deployment
kubectl create -f https://raw.githubusercontent.com/lwolf/kube-cleanup-operator/master/deploy/deployment.yaml


kubectl logs -f $(kubectl get pods --namespace default -l "run=cleanup-operator" -o jsonpath="{.items[0].metadata.name}")

# Use simple job to test it
kubectl create -f https://raw.githubusercontent.com/kubernetes/kubernetes.github.io/master/docs/concepts/workloads/controllers/job.yaml
```

## Docker images

```docker pull lwolf/kube-cleanup-operator```

```docker pull quay.io/lwolf/kube-cleanup-operator```

## Development

```
$ make install_deps
$ make build
$ ./bin/kube-cleanup-operator --help
Usage of ./bin/kube-cleanup-operator:
  -namespace string
    	Watch only this namespaces
  -run-outside-cluster
    	Set this flag when running outside of the cluster.
  -keep-successful
        the number of hours to keep a succesfull job
        -1 - forever 
        0  - never (default)
        >0 - number of hours
  -keep-failures
        the number of hours to keep a succesfull job
        -1 - forever (default)
        0  - never
        >0 - number of hours
  -keep-pending
        the number of hours to keep a pending job
        -1 - forever (default)
        0  - forever
        >0 - number of hours
  -dry run
        Perform dry run, print only
        
$ ./bin/kube-cleanup-operator --run-outside-cluster --namespace=default --keep-successful=0 --keep-failure=-1 --keep-pending=-1
```
