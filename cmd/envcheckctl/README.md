# envcheckctl

The application `envcheckctl` is used to inspect a Kubernetes/OpenShift cluster for common agent
installation issues.

## Pull Debug Data

The application `envcheckctl` is capable of collecting data directly from the cluster.

```bash
# use a specific kubeconfig file
$ envcheckctl inspect -kubeconfig $KUBECONFIG

# use the default context for kubectl
$ envcheckctl inspect
```
As a result of the data pull, the application generates a JSON file with the name like `cluster-info-${TIMESTAMP}.json`.
The application also populates standard output with the following information:

```yaml
# Summary information of all resources found
pods=256, running=256, nodes=19, containers=338, namespaces=13, deployments=56, replicaSets=56, daemonsets=9, statefulsets=7, duration=1.435853996s

# Coverage indicates how many hosts in the cluster is actively monitored by Instana. Generally we expect this to be 100% however it is common to have less than 100% with OpenShift and self-managed Kubernetes clusters whereby the control-plane is not monitored due to taints. Less than 100% coverage in the absence of a taint can be an indicator for broken traces and missing infrastructure metrics.
coverage
- "13 of 19 (68.42%)"

# distribution type kubernetes / openshift / eks / gke / aks
serverDistribution
 - eks

# Server version is useful for indicating API compatibility and can be used to quickly identify any release related compatibility issues such as deprecations.
serverVersion
 - v1.23.14-eks-ffeb93d

# Top 10 agents ordered by number of restarts. This is useful for diagnosing whether any agents in the cluster are experiencing frequent restarts.
agentRestarts
- "instana-agent-4j8zq"=5
- "instana-agent-jt84d"=5
- "instana-agent-t5tcs"=4
- "instana-agent-fvgbj"=2
- "instana-agent-jvbhr"=1

# Agent status indicates the current state of all Instana agents in the cluster. If any agents are not Running then it indicative of potentially anomalous infrastructure metric and trace behaviour.
agentStatus
- "Running"=13

# Lists the distribution of helm chart versions and number of pods
chartVersions
- "1.2.45"=13

# Distribution of cniPlugin
cniPlugins
- "cilium"=19

# Distribution of container Runtimes
containerRuntimes
- "containerd://1.6.6"=19

# Distribution of nodes types
instanceTypes
- "c5.12xlarge"=4
- "m5.4xlarge"=9
- "m5.large"=3
- "m5.xlarge"=3

# Distribution of kernels on nodes
kernels
- "5.4.231-137.341.amzn2.x86_64"=19

# Distribution of kubelets on nodes
kubelet
- "v1.22.17-eks-48e63af"=19

# Distribution of os images on nodes
osImages
- "Amazon Linux 2"=19

# Distribution of pods statuses
podStatus
- "Running"=256

# Distribution of proxy versions on nodes
proxy
- "v1.22.17-eks-48e63af"=19

# Distribution of zones that nodes belong, this can be used to determine which zone has an outage 
zones
- "eu-central-1a"=6
- "eu-central-1b"=5
- "eu-central-1c"=8

# Distribution of configmaps linked to pods
linkedConfigMaps
- "instana-agent/instana-agent"=13

# Distribution of pods to owner types
owners
- "DaemonSet"=155
- "ReplicaSet"=88
- "StatefulSet"=13
- "Unknown"=1
- "Standalone"=2

# Resource limit information for instana-agent
sizing=instana-agent cpurequests=500m cpulimits=1.5 memoryrequests=512Mi memorylimits=512Mi heap=170M
```

## Load Debug Data
The json data file can be loaded using the following instruction:

```bash
envcheckctl inspect -podfile=cluster-info-1672531200.json
```
Here is the snippet of the data from the debug data file:

```json
{
  "Name": "https://cluster.name.com",
  "NodeCount": 19,
  "Nodes": [
    {
      "ContainerRuntime": "containerd://1.1.1",
      "InstanceType": "c5.12xlarge",
      "KernelVersion": "5.4.231-137.341.amzn2.x86_64",
      "KubeletVersion": "v1.22.17-eks-48e63af",
      "Name": "ip-10-10-10-10.zone.compute.internal",
      "OSImage": "Amazon Linux 2",
      "ProxyVersion": "v1.22.17-eks-48e63af",
      "Zone": "zone_name"
    },
    ...
    ...
  ],
  "PodCount": 256,
  "Pods": [
    {
      "ChartVersion": "v1.9.1",
      "Containers": [
        {
          "Name": "cert-manager",
          "Image": "your_image_repo:cert-manager-controller_v1.1.1"
        },
        ...
        ...
      ],
      "LinkedConfigMaps": [
        {
          "Name": "cert-manager-common-configd",
          "Namespace": "cert-manager"
        },
        ...
        ...
      ],
      "Host": "10.10.10.10",
      "IsRunning": true,
      "Name": "cert-manager-xyz-abc",
      "Namespace": "cert-manager",
      "Owners": {
        "cert-manager-699cd85758": "ReplicaSet"
      },
      "Restarts": 0,
      "Status": "Running"
    },
    ...
    ...
  ],
  "Version": "",
  "Started": "2023-03-17T10:05:53.991188919+01:00",
  "Finished": "2023-03-17T10:05:55.427042915+01:00"
}
```

