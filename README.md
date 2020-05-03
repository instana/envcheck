envcheck
========

[![Godoc](https://godoc.org/github.com/instana/envcheck?status.svg)](https://godoc.org/github.com/instana/envcheck) [![Go Report Card](https://goreportcard.com/badge/github.com/instana/envcheck)](https://goreportcard.com/report/github.com/instana/envcheck)

Used to inspect a Kubernetes/OpenShift cluster for common agent
 installation issues. The memory and CPU requirements are minimal overhead
 so that it can run with little concern in a cluster.

Current Capabilities
--------------------

 * daemon/pinger - Validate connectivity from namespace/pod to local host network.
 * envcheckctl - Pull a dump of all pods in the cluster .
 * envcheckctl - Add agent memory sizing guide for a K8S cluster.

Future Capabilities
-------------------

 * Add instana-agent config map to the JSON dump.
 * Check access to backend from all daemonsets.
 * Check API permissions.
 * Aggregate and collect all metrics with a coordinator.

Requirements
------------

- Docker for Desktop.
- Make.
- Kubernetes or OpenShift cluster.

Installation
------------

Before installing, update the YAML file image keys with your docker repo.

### Daemon and Pinger

The daemon and pinger validate connectivity in your cluster between the agent and an instrumented application.

The commands below will build and deploy the daemon and pinger to your cluster.

```shell
export DOCKER_REPO=YOUR_DOCKER_REPO_URL
make # this will 
kubectl create namespace instana-agent
# deploy daemon pods that bind to host network similar to the agent.
kubectl apply -f base/daemon.yaml
# watch for all daemon pods to get to a running state.
kubectl get pods -n instana-agent -l name=envchecker -w
# deploy pinger pods to default namespace
kubectl apply -f base/pinger.yaml
# watch logs for connectivity status
kc logs -l name=pinger -n default -f
```

If the pinger is able to communicate with the same host daemonset you'll see
a log entry similar to:
```
2020/04/29 19:46:41 ping=success pod=default/pinger-87466 address=192.168.253.102:42699
```

If it is unable to communicate with the same host daemonset you'll see a log
entry similar to:

```
2020/04/29 20:59:30 ping=failure pod=default/pinger-v7xvb address=192.168.253.101:42699 err='Get "http://192.168.253.101:42699/ping": dial tcp 192.168.253.101:42699: i/o timeout'
```

The logs for the daemonset should be relatively quiet. This is an example log
output for a small cluster:
```
kc logs -n instana-agent -l name=envchecker -f
2020/04/29 19:46:06 daemon=7c09e63 listen=0.0.0.0:42699 pod=instana-agent/envchecker-4g7bt podIP=192.168.253.101 nodeIP=192.168.253.101
2020/04/29 19:46:06 daemon=7c09e63 listen=0.0.0.0:42699 pod=instana-agent/envchecker-9qggf podIP=192.168.253.102 nodeIP=192.168.253.102
```

### envcheckctl

The application envcheckctl is capable of collecting data to aid in debugging a cluster.

```shell
# use a specific kubeconfig file
$ envcheckctl -kubeconfig $KUBECONFIG
# ...

# use the default context for kubectl
$ envcheckctl
# cluster connection
2020/05/02 19:33:47 envcheckctl=997fe30, cluster=https://88fe4a1b-f913-432f-bb03-64c6fcda31dd.k8s.ondigitalocean.com, start=2020-05-02T19:33:47-03:00
2020/05/02 19:33:47 Collecting pod details. Duration varies depending on the cluster.
# cluster summary
2020/05/02 19:33:48 pods=33, running=33, nodes=3, containers=36, namespaces=3, deployments=17, daemonsets=5, statefulsets=0, duration=955.355516ms
# suggested agent sizing
2020/05/02 19:33:48 sizing=instana-agent cpurequests=500m cpulimits=1.5 memoryrequests=512Mi memorylimits=512Mi heap=170M
```

Debug Container
---------------

```shell
 # launch a debug container in the default namespace
kubectl run -it --rm --restart=Never alpine --image=alpine sh

 # http request to the ping end-point for a node
wget -q -S -O - http://${NODE_IP}:42699/ping && echo ''
```

