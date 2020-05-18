package cluster

import (
	"time"
)

// PodApplyable is the interface to receive pod info from a pod collection.
type PodApplyable interface {
	Each(PodInfo)
}

// Info is a data structure for relevant cluster data.
type Info struct {
	Name     string
	PodCount int
	Pods     []PodInfo
	Version  string
	Started  time.Time
	Finished time.Time
}

// Apply iterates over each pod and yields it to the list of applyables.
func (info *Info) Apply(applyable ...PodApplyable) {
	for _, pod := range info.Pods {
		for _, a := range applyable {
			a.Each(pod)
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
}

// ContainerInfo is summary details for a container.
type ContainerInfo struct {
	Name  string
	Image string
}
