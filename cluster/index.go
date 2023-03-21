package cluster

import (
	"fmt"
	"strconv"
)

// NewIndex builds a new empty index for PodInfo.
func NewIndex() *Index {
	return &Index{
		CNIPlugins:        make(Counter),
		Containers:        make(Set),
		DaemonSets:        make(Set),
		Deployments:       make(Set),
		Namespaces:        make(Set),
		Nodes:             make(Set),
		Pods:              make(Set),
		Running:           make(Set),
		StatefulSets:      make(Set),
		AgentRestarts:     make(Counter),
		AgentStatus:       make(Counter),
		ChartVersions:     make(Counter),
		ContainerRuntimes: make(Counter),
		InstanceTypes:     make(Counter),
		KernelVersions:    make(Counter),
		KubeletVersions:   make(Counter),
		LinkedConfigMaps:  make(Counter),
		OSImages:          make(Counter),
		Owners:            make(Counter),
		PodStatus:         make(Counter),
		ProxyVersions:     make(Counter),
		Zones:             make(Counter),
	}
}

// Index provides indexes for a number of the cluster entities.
type Index struct {
	Containers        Set
	DaemonSets        Set
	Deployments       Set
	Namespaces        Set
	Nodes             Set
	Pods              Set
	Running           Set
	StatefulSets      Set
	AgentRestarts     Counter
	AgentStatus       Counter
	ChartVersions     Counter
	CNIPlugins        Counter
	ContainerRuntimes Counter
	InstanceTypes     Counter
	KernelVersions    Counter
	KubeletVersions   Counter
	LinkedConfigMaps  Counter
	OSImages          Counter
	ProxyVersions     Counter
	Zones             Counter
	PodStatus         Counter
	Owners            Counter
}

// Summary provides a summary overview of the number of entities in the cluster.
type Summary struct {
	Containers   int
	DaemonSets   int
	Deployments  int
	Namespaces   int
	Nodes        int
	Pods         int
	Running      int
	StatefulSets int
}

// Summary provides a summary pods for all the entities.
func (index *Index) Summary() Summary {
	return Summary{
		Containers:   index.Containers.Len(),
		DaemonSets:   index.DaemonSets.Len(),
		Deployments:  index.Deployments.Len(),
		Nodes:        index.Nodes.Len(),
		Namespaces:   index.Namespaces.Len(),
		Pods:         index.Pods.Len(),
		Running:      index.Running.Len(),
		StatefulSets: index.StatefulSets.Len(),
	}
}

const (
	// DaemonSet is the related owner key for that type of k8s entity.
	DaemonSet = "DaemonSet"
	// ReplicaSet is the related owner key for that type of k8s entity.
	ReplicaSet = "ReplicaSet"
	// StatefulSet is the related owner key for that type of k8s entity.
	StatefulSet = "StatefulSet"
)

func (index *Index) EachNode(node NodeInfo) {
	index.ContainerRuntimes.Add(node.ContainerRuntime)
	index.InstanceTypes.Add(node.InstanceType)
	index.KernelVersions.Add(node.KernelVersion)
	index.KubeletVersions.Add(node.KubeletVersion)
	index.OSImages.Add(node.OSImage)
	index.ProxyVersions.Add(node.ProxyVersion)
	index.Zones.Add(node.Zone)
}

// EachPod extracts the relevant pod details and integrates it into the index.
func (index *Index) EachPod(pod PodInfo) {
	qualifiedName := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	index.Pods.Add(qualifiedName)
	if pod.IsRunning {
		index.Running.Add(qualifiedName)
	}
	index.Namespaces.Add(pod.Namespace)
	index.PodStatus.Add(pod.Status)

	if pod.Host != "" {
		index.Nodes.Add(pod.Host)
	}

	for i, c := range pod.Containers {
		var name = c.Name
		if name == "" {
			name = strconv.Itoa(i)
		}
		index.Containers.Add(fmt.Sprintf("%s/%s", qualifiedName, name))
	}

	for n, t := range pod.Owners {
		switch t {
		case DaemonSet:
			if IsCNIPlugin(n) {
				index.CNIPlugins.Add(n)
			}
			if IsInstanaAgent(pod) {
				index.AgentRestarts.Set(pod.Name, pod.Restarts)
				index.AgentStatus.Add(pod.Status)
				index.ChartVersions.Add(pod.ChartVersion)
				for _, cm := range pod.LinkedConfigMaps {
					index.LinkedConfigMaps.Add(fmt.Sprintf("%s/%s", cm.Namespace, cm.Name))
				}
			}
			index.DaemonSets.Add(n)

		case ReplicaSet: // hackish way to calculate deployments
			index.Deployments.Add(n)

		case StatefulSet:
			index.StatefulSets.Add(n)
		}

		ownerType := t
		if ownerType == "" {
			ownerType = "Unknown"
		}
		index.Owners.Add(ownerType)
	}

	if len(pod.Owners) == 0 {
		index.Owners.Add("Standalone")
	}
}

func IsInstanaAgent(pod PodInfo) bool {
	for _, c := range pod.Containers {
		if c.Name == "instana-agent" {
			return true
		}
	}
	return false
}

func IsCNIPlugin(n string) bool {
	m := map[string]bool{
		"aws-node":    true,
		"calico":      true,
		"cilium":      true,
		"flannel":     true,
		"kube-router": true,
		"multus":      true,
		"sdn":         true,
	}

	return m[n]
}

type Counter map[string]int

func (c Counter) Add(item string) {
	i := c[item]
	i++
	c[item] = i
}

func (c Counter) Set(item string, value int) {
	c[item] = value
}

func (c Counter) Len() int {
	return len(c)
}

// Set provides a set collection for strings.
type Set map[string]bool

// Add integrates the item into the underlying set.
func (s Set) Add(item string) {
	s[item] = true
}

// Len lists the number of items in the set.
func (s Set) Len() int {
	return len(s)
}
