package cluster

import (
	"fmt"
	"strconv"
)

// IndexFrom creates a new cluster index for relevant cluster entities.
func IndexFrom(info *Info) *Index {
	index := NewIndex()

	info.Apply(index)

	return index
}

func NewIndex() *Index {
	return &Index{
		Containers:   make(Set),
		DaemonSets:   make(Set),
		Deployments:  make(Set),
		Namespaces:   make(Set),
		Nodes:        make(Set),
		Pods:         make(Set),
		Running:      make(Set),
		StatefulSets: make(Set),
	}
}

// Index provides indexes for a number of the cluster entities.
type Index struct {
	Containers   Set
	DaemonSets   Set
	Deployments  Set
	Namespaces   Set
	Nodes        Set
	Pods         Set
	Running      Set
	StatefulSets Set
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

// Summary provides a summary count for all of the entities.
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
	DaemonSet   = "DaemonSet"
	ReplicaSet  = "ReplicaSet"
	StatefulSet = "StatefulSet"
)

// Each extracts the relevant pod details and integrates it into the index.
func (index *Index) Each(pod PodInfo) {
	qualifiedName := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
	index.Pods.Add(qualifiedName)
	if pod.IsRunning {
		index.Running.Add(qualifiedName)
	}
	index.Namespaces.Add(pod.Namespace)
	index.Nodes.Add(pod.Host)

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

// Contains tests if the item is found in the set.
func (s Set) Contains(item string) bool {
	_, present := s[item]
	return present
}
