package cluster

import (
	"fmt"
	"strconv"
	"strings"
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
		ContainerRuntimes: make(Counter),
		InstanceTypes:     make(Counter),
		KernelVersions:    make(Counter),
		KubeletVersions:   make(Counter),
		OSImages:          make(Counter),
		ProxyVersions:     make(Counter),
		Zones:             make(Counter),
	}
}

// Index provides indexes for a number of the cluster entities.
type Index struct {
	CNIPlugins        Counter
	Containers        Set
	DaemonSets        Set
	Deployments       Set
	Namespaces        Set
	Nodes             Set
	Pods              Set
	Running           Set
	StatefulSets      Set
	ContainerRuntimes Counter
	InstanceTypes     Counter
	KernelVersions    Counter
	KubeletVersions   Counter
	OSImages          Counter
	ProxyVersions     Counter
	Zones             Counter
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
			index.DaemonSets.Add(n)
			break
		case ReplicaSet: // hackish way to calculate deployments
			index.Deployments.Add(n)
			break
		case StatefulSet:
			index.StatefulSets.Add(n)
		}
	}
}

func IsCNIPlugin(n string) bool {
	if n == "aws-node" {
		return true
	}
	if strings.HasPrefix(n, "cilium") {
		return true
	}
	if strings.HasPrefix(n, "calico") {
		return true
	}
	if strings.HasPrefix(n, "flannel") {
		return true
	}
	if strings.HasPrefix(n, "kube-router") {
		return true
	}
	return false
}

type Counter map[string]int

func (c Counter) Add(item string) {
	i := c[item]
	i++
	c[item] = i
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
