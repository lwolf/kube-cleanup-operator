# Kubernetes cleanup operator

Experimental Kubernetes Operator to automatically delete completed Jobs and their Pods.
Controller listens for changes in Pods created by Jobs and deletes it on Completion.

Some defaults:
* All Namespaces are monitored by default
* Only Pods created by Jobs are monitored
* Only Pods in Completed state with 0 restarts are deleted


## Usage

```
# remember to change namespace in RBAC manifests for monitoring namespaces other than "default"

kubectl create -f https://raw.githubusercontent.com/lwolf/kube-cleanup-operator/master/deploy/rbac.yaml

# create deployment
kubectl create -f https://raw.githubusercontent.com/lwolf/kube-cleanup-operator/master/deploy/deployment.yaml


kubectl logs -f <operator-pod>

# Use simple job to test it
kubectl create -f https://raw.githubusercontent.com/kubernetes/kubernetes.github.io/
```


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

$ ./bin/kube-cleanup-operator --run-outside-cluster --namespace=default
```
