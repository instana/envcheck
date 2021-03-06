envcheck ![status: Beta](https://img.shields.io/badge/Status-BETA-YELLOW.svg)
=============================================================================

[![Godoc](https://godoc.org/github.com/instana/envcheck?status.svg)](https://godoc.org/github.com/instana/envcheck) [![Go Report Card](https://goreportcard.com/badge/github.com/instana/envcheck)](https://goreportcard.com/report/github.com/instana/envcheck) [![CodeBuild Badge](https://codebuild.us-west-2.amazonaws.com/badges?uuid=eyJlbmNyeXB0ZWREYXRhIjoialJ0L0lFUlFraEJKNU1tYVcwcDZWN3d4M2lJMjZTM003TG9OYXZOVndlSXNxQnlQeGt4NjVQUmpRa3pqcUdnajcrLzd3MWtxYnkyckpDWmFHT2ZMMVBnPSIsIml2UGFyYW1ldGVyU3BlYyI6IksyckVKVXc0V2NoYkRxQ0giLCJtYXRlcmlhbFNldFNlcmlhbCI6MX0%3D&branch=master)](https://us-west-2.console.aws.amazon.com/codesuite/codebuild/projects/envcheck/history) ![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/instana/envcheck) ![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/instana/envcheck) [![Codecov](https://img.shields.io/codecov/c/gh/instana/envcheck)](https://codecov.io/gh/Instana/envcheck)

Used to inspect a Kubernetes/OpenShift cluster for common agent
 installation issues. The memory and CPU requirements are minimal overhead
 so that it can run with little concern in a cluster.

Overview
--------

The following components are provided by this repository:

- **envcheckctl** - CLI tool that remotely inspects a cluster and dump to JSON.
- **envcheckctl** - load JSON dump for offline inspection.
- **pinger** - in cluster service that verifies connectivity from a specified
  namespace to the Instana agents namespace.
- **daemon** - in cluster service that binds in the Instana agent namespace.

Current Capabilities
--------------------

 * **daemon/pinger** - Validate connectivity from namespace/pod to local host
   network.
 * **envcheckctl** - Pull a dump of all pods in the cluster .
 * **envcheckctl** - Add agent memory sizing guide for a K8S cluster.
 * **envcheckctl** - Find k8s leader.

Future Capabilities
-------------------

 * [x] Find k8s leader.
 * [ ] Inject a profiler into a pod.
 * [ ] Add instana-agent config map to the JSON dump.
 * [ ] Check access to backend from all daemonsets.
 * [ ] Check API permissions.
 * [ ] Aggregate and collect all metrics with a coordinator.
 * [ ] Report presence of service meshes and CNI details.
 * [ ] Check for presence of Pod Security Policy.
 * [x] Check for events on instana-agent DaemonSet.

Install Requirements
--------------------

- Cluster Admin access to Kubernetes/OpenShift cluster.
- kubectl or OpenShift client.
- latest [envcheckctl](https://github.com/instana/envcheck/releases/latest)
  binary for your OS.

### Running envcheckctl

#### Pull Debug Data

The application envcheckctl is capable of collecting data to aid in debugging a
 cluster.

```bash
# use a specific kubeconfig file
$ envcheckctl inspect -kubeconfig $KUBECONFIG
# ...

# use the default context for kubectl
$ envcheckctl inspect
# cluster connection
2020/05/02 19:33:47 envcheckctl=997fe30, cluster=https://88fe4a1b-f913-432f-bb03-64c6fcda31dd.k8s.ondigitalocean.com, start=2020-05-02T19:33:47-03:00
2020/05/02 19:33:47 Collecting pod details. Duration varies depending on the cluster.
# cluster summary
2020/05/02 19:33:48 pods=33, running=33, nodes=3, containers=36, namespaces=3, deployments=17, daemonsets=5, statefulsets=0, duration=955.355516ms
# suggested agent sizing
2020/05/02 19:33:48 sizing=instana-agent cpurequests=500m cpulimits=1.5 memoryrequests=512Mi memorylimits=512Mi heap=170M
```

#### Load Debug Data

```bash
# load dump from disk
envcheckctl inspect -podfile=cluster-info-1589217975.json
# cluster info
2020/05/14 11:54:58 envcheckctl=, cluster=https://192.168.253.100:6443, podfile=cluster-info-1589217975.json
# note: reported duration is the duration of the original query and not load time.
2020/05/14 11:54:58 pods=17, running=17, nodes=3, containers=17, namespaces=4, deployments=5, daemonsets=3, statefulsets=0, duration=20.66ms
2020/05/14 11:54:58 sizing=instana-agent cpurequests=500m cpulimits=1.5 memoryrequests=512Mi memorylimits=512Mi heap=170M
```

#### Profile Pod (Under development)

```bash
# print the leader pod as defined by the instana end-point
envcheckctl profile -leader

# profile the Instana k8s leader
envcheckctl profile -namespace=instana-agent-2 -leader -profile=profile.tgz
# outputs profile-instana-agent-2-instana-agent-x1z2a-${TS}.tgz

# profile arbitrary pod
envcheckctl profile -namespace=default -pod=mypod-x1z2a -profile
# outputs profile-default-mypod-x1z2a-${TS}.tgz
```

### Running Daemon

```bash
# optional only required if agent not installed.
kubectl create namespace instana-agent
# deploy daemon pods that bind to host network similar to the agent.
envcheckctl daemon
# wait for all pods to have a status of Running without few if any restarts.
kubectl get pods -l app.kubernetes.io/name=envchecker -n instana-agent -w
```

The logs for the daemonset should be quiet with one log line per node. 
This is the log output for a small 3 node cluster. **Note** by default it does
 not run a pod on the master nodes.
```
kubectl logs -n instana-agent -l app.kubernetes.io/name=envchecker -f
2020/04/29 19:46:06 daemon=7c09e63 listen=0.0.0.0:42699 pod=instana-agent/envchecker-4g7bt podIP=192.168.253.101 nodeIP=192.168.253.101
2020/04/29 19:46:06 daemon=7c09e63 listen=0.0.0.0:42699 pod=instana-agent/envchecker-9qggf podIP=192.168.253.102 nodeIP=192.168.253.102
```

### Running Pinger

```bash
# install the pinger to the default namespace.
envcheckctl ping
# wait for all pods to have a status of Running. There should be no restarts.
kubectl get pods -l app.kubernetes.io/name=pinger -n default -w

# install pinger to another namespace and ping the daemon on the specified host
envcheckctl ping -ns=other-namespace -pinghost=localhost
# wait for all pods to have a status of Running. There should be no restarts.
kubectl get pods -l app.kubernetes.io/name=pinger -n other-namespace -w
```

Logs can be retrieved with the following command:

```bash
kubectl logs -l app.kubernetes.io/name=pinger -n $NAMESPACE -f
```

If the pinger is able to communicate with the same host daemonset the logs
should contain an entry prefixed with `ping=success` as follows:

```bash
2020/04/29 19:46:41 ping=success pod=default/pinger-87466 address=192.168.253.102:42699
```

If it is unable to communicate with the same host daemon the logs should contain
an entry prefixed with `ping=failure` as follows:

```bash
2020/04/29 20:59:30 ping=failure pod=default/pinger-v7xvb address=192.168.253.101:42699 err='Get "http://192.168.253.101:42699/ping": dial tcp 192.168.253.101:42699: i/o timeout'
```

Build Requirements
------------------

- Docker for Desktop.
- Make.
- Golang 1.14+

Building
--------

**Note** if building from source the YAML files in base require the container 
image to reflect the relevant Docker repository.

### Building Daemon and Pinger

The commands below will build and publish the daemon and pinger docker images to
 the repository specified by the environment variable `$DOCKER_REPO`.

```bash
export DOCKER_REPO=YOUR_DOCKER_REPO_URL
make publish # build and publish the docker images
```

### Building envcheckctl

The command below will build 3 binaries that can be executed on these platforms:

 - Linux (envcheckctl.amd64)
 - OSX (envcheckctl.darwin64)
 - Windows (envcheckctl.exe)

```bash
make envcheckctl
```

Debug Container
---------------

```shell
 # launch a debug container in the default namespace
kubectl run -it --rm --restart=Never alpine --image=alpine sh

 # http request to the ping end-point for a node
wget -q -S -O - http://${NODE_IP}:42699/ping && echo ''
```
