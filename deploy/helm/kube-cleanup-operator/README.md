# kube-cleanup-operator

![Version: 1.0.0](https://img.shields.io/badge/Version-1.0.0-informational?style=flat-square) ![AppVersion: 0.8.1](https://img.shields.io/badge/AppVersion-0.8.1-informational?style=flat-square)

Kubernetes Operator to automatically delete completed Jobs and their Pods

## Source Code

* <https://github.com/lwolf/kube-cleanup-operator.git>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` |  |
| annotations | object | `{}` |  |
| args[0] | string | `"--namespace=default"` |  |
| args[1] | string | `"--delete-successful-after=5m"` |  |
| args[2] | string | `"--delete-failed-after=120m"` |  |
| args[3] | string | `"--delete-pending-pods-after=60m"` |  |
| args[4] | string | `"--delete-evicted-pods-after=60m"` |  |
| args[5] | string | `"--delete-orphaned-pods-after=60m"` |  |
| args[6] | string | `"--legacy-mode=false"` |  |
| containerSecurityContext | string | `nil` |  |
| envVariables | list | `[]` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"quay.io/lwolf/kube-cleanup-operator"` |  |
| image.tag | string | `"latest"` |  |
| labels | object | `{}` |  |
| livenessProbe.httpGet.path | string | `"/metrics"` |  |
| livenessProbe.httpGet.port | int | `7000` |  |
| nodeSelector | object | `{}` |  |
| podAnnotations | object | `{}` |  |
| podLabels | object | `{}` |  |
| priorityClassName | string | `nil` |  |
| rbac.create | bool | `true` |  |
| readinessProbe.failureThreshold | int | `3` |  |
| readinessProbe.httpGet.path | string | `"/metrics"` |  |
| readinessProbe.httpGet.port | int | `7000` |  |
| readinessProbe.initialDelaySeconds | int | `5` |  |
| readinessProbe.periodSeconds | int | `30` |  |
| readinessProbe.timeoutSeconds | int | `5` |  |
| replicas | int | `1` |  |
| resources.limits.cpu | string | `"50m"` |  |
| resources.limits.memory | string | `"64Mi"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| securityContext | string | `nil` |  |
| service.annotations."prometheus.io/port" | string | `"7000"` |  |
| service.annotations."prometheus.io/scrape" | string | `"true"` |  |
| service.labels | object | `{}` |  |
| service.port | int | `80` |  |
| service.type | string | `"ClusterIP"` |  |
| serviceAccount.create | bool | `true` |  |
| serviceAccount.name | string | `nil` |  |
| serviceMonitor.annotations | object | `{}` |  |
| serviceMonitor.enabled | bool | `false` |  |
| serviceMonitor.labels | object | `{}` |  |
| serviceMonitor.scrapeInterval | string | `"10s"` |  |
| strategy.type | string | `"RollingUpdate"` |  |
| tolerations | list | `[]` |  |