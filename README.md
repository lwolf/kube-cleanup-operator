# Kubernetes cleanup operator

[![Build Status](https://travis-ci.org/lwolf/kube-cleanup-operator.svg?branch=master)](https://travis-ci.org/lwolf/kube-cleanup-operator)
[![Go Report Card](https://goreportcard.com/badge/github.com/lwolf/kube-cleanup-operator)](https://goreportcard.com/report/github.com/lwolf/kube-cleanup-operator)
[![Docker Repository on Quay](https://quay.io/repository/lwolf/kube-cleanup-operator/status "Docker Repository on Quay")](https://quay.io/repository/lwolf/kube-cleanup-operator)
[![codecov](https://codecov.io/gh/lwolf/kube-cleanup-operator/branch/master/graph/badge.svg)](https://codecov.io/gh/lwolf/kube-cleanup-operator)

Kubernetes Controller to automatically delete completed Jobs and Pods.
Controller listens for changes in Pods and Jobs and acts accordingly with config arguments.

Some common use-case scenarios:
* Delete Jobs and their pods after their completion
* Delete Pods stuck in a Pending state
* Delete Pods in Evicted state
* Delete orphaned Pods (Pods without an owner in non-running state)

| flag name                  | pod                                                   | job                           |
| -------------------------- | ----------------------------------------------------- | ----------------------------- |
| delete-successful-after    | delete after specified period if owned by the job     | delete after specified period |
| delete-failed-after        | delete after specified period if owned by the job     | delete after specified period |
| delete-orphaned-pods-after | delete after specified period (any completion status) | N/A                           |
| delete-evicted-pods-after  | delete on discovery                                   | N/A                           |
| delete-pending-pods-after  | delete after specified period                         | N/A                           |


## Helm chart

Chart is available to install from https://charts.lwolf.org/ (https://github.com/lwolf/kube-charts)

```
$ helm repo add lwolf-charts http://charts.lwolf.org
"lwolf-charts" has been added to your repositories
$ helm search kube-cleanup
NAME                              	CHART VERSION	APP VERSION	DESCRIPTION
lwolf-charts/kube-cleanup-operator	1.0.0        	v0.8.1     	Kubernetes Operator to automatically delete completed Job...
```


## Usage

![screensharing](http://g.recordit.co/aDU52FJIwP.gif)

```
# remember to change namespace in RBAC manifests for monitoring namespaces other than "default"

kubectl create -f https://raw.githubusercontent.com/lwolf/kube-cleanup-operator/master/deploy/deployment/rbac.yaml

# create deployment
kubectl create -f https://raw.githubusercontent.com/lwolf/kube-cleanup-operator/master/deploy/deployment/deployment.yaml


kubectl logs -f $(kubectl get pods --namespace default -l "run=cleanup-operator" -o jsonpath="{.items[0].metadata.name}")

# Use simple job to test it
kubectl create -f https://k8s.io/examples/controllers/job.yaml
```


## Docker images

```docker pull quay.io/lwolf/kube-cleanup-operator```

or you can build it yourself as follows:

```console
$ docker build .
```

## Development

```console
$ make install_deps
$ make build
$ ./bin/kube-cleanup-operator -run-outside-cluster -dry-run=true
```

## Usage

Pre v0.7.0

```
    $ ./bin/kube-cleanup-operator --help
    Usage of ./bin/kube-cleanup-operator:
      -namespace string
            Watch only this namespace (omit to operate clusterwide)
      -run-outside-cluster
            Set this flag when running outside of the cluster.
      -keep-successful
            the number of hours to keep a successful job
            -1 - forever 
            0  - never (default)
            >0 - number of hours
      -keep-failures
            the number of hours to keep a failed job
            -1 - forever (default)
            0  - never
            >0 - number of hours
      -keep-pending
            the number of hours to keep a pending job
            -1 - forever (default)
            0  - forever
            >0 - number of hours
      -dry-run
            Perform dry run, print only
``` 

After v0.7.0

```
Usage of ./bin/kube-cleanup-operator:
  -delete-evicted-pods-after duration
        Delete pods in evicted state (golang duration format, e.g 5m), 0 - never delete (default 15m0s)
  -delete-failed-after duration
        Delete jobs and pods in failed state after X duration (golang duration format, e.g 5m), 0 - never delete
  -delete-orphaned-pods-after duration
        Delete orphaned pods. Pods without an owner in non-running state (golang duration format, e.g 5m), 0 - never delete (default 1h0m0s)
  -delete-pending-pods-after duration
        Delete pods in pending state after X duration (golang duration format, e.g 5m), 0 - never delete
  -delete-successful-after duration
        Delete jobs and pods in successful state after X duration (golang duration format, e.g 5m), 0 - never delete (default 15m0s)
  -dry-run
        Print only, do not delete anything.
  -ignore-owned-by-cronjobs
        [EXPERIMENTAL] Do not cleanup pods and jobs created by cronjobs
  -keep-failures int
        Number of hours to keep failed jobs, -1 - forever (default) 0 - never, >0 number of hours (default -1)
  -keep-pending int
        Number of hours to keep pending jobs, -1 - forever (default) >0 number of hours (default -1)
  -keep-successful int
        Number of hours to keep successful jobs, -1 - forever, 0 - never (default), >0 number of hours
  -legacy-mode true
        Legacy mode: true - use old `keep-*` flags, `false` - enable new `delete-*-after` flags (default true)
  -listen-addr string
        Address to expose metrics. (default "0.0.0.0:7000")
  -namespace string
        Limit scope to a single namespace
  -run-outside-cluster
        Set this flag when running outside of the cluster.
```

### Optional parameters 

DISCLAIMER: These parameters are not supported on this project since they are implemented by the underlying libraries. Any malfunction regarding the use them is not covered by this GitHub repository. They are included in this documentation since the debugging process is simplified.

```
-alsologtostderr
  log to standard error as well as files
-log_backtrace_at value
  when logging hits line file:N, emit a stack trace
-log_dir string
  If non-empty, write log files in this directory
-logtostderr
  log to standard error instead of files
-vmodule value
  comma-separated list of pattern=N settings for file-filtered logging
```
