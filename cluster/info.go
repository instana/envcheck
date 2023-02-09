package cluster

import (
	"time"
)

// Applyable is the interface to receive pod info from a pod collection.
type Applyable interface {
	EachPod(PodInfo)
	EachNode(NodeInfo)
}

// Info is a data structure for relevant cluster data.
type Info struct {
	Name      string
	NodeCount int
	Nodes     []NodeInfo
	PodCount  int
	Pods      []PodInfo
	Version   string
	Started   time.Time
	Finished  time.Time
}

// Apply iterates over each pod and yields it to the list of applyables.
func (info *Info) Apply(applyable ...Applyable) {
	for _, pod := range info.Pods {
		for _, a := range applyable {
			a.EachPod(pod)
		}
	}

	for _, node := range info.Nodes {
		for _, a := range applyable {
			a.EachNode(node)
		}
	}
}

// PodInfo is summary details for a pod.
type PodInfo struct {
	Containers []ContainerInfo
	Host       string
	IsRunning  bool
	Name       string
	Namespace  string
	Owners     map[string]string
	Restarts   int
}

// ContainerInfo is summary details for a container.
type ContainerInfo struct {
	Name  string
	Image string
}
