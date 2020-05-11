package cluster

import (
	"context"
	"fmt"
	"strconv"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	// imports all auth methods for kubernetes go client.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

// Query is a query interface for the cluster.
type Query interface {
	// AllPods returns the list of pods from the related cluster.
	AllPods() ([]PodInfo, error)
	Host() string
}

// New builds a new KubernetesQuery implementation with the given kubeconfig.
func New(kubeconfig string) (*KubernetesQuery, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &KubernetesQuery{config.Host, clientset}, nil
}

// KubernetesQuery is a concrete Kubernetes client to query various cluster info.
type KubernetesQuery struct {
	host string
	*kubernetes.Clientset
}

// Host provides the host info for the cluster.
func (q *KubernetesQuery) Host() string {
	return q.host
}

// AllPods retrieves all pod info from the cluster.
func (q *KubernetesQuery) AllPods() ([]PodInfo, error) {
	var cont string
	var podList []PodInfo
	namespaces := make(map[string]bool)

	for true {
		pods, err := q.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{Limit: 100, Continue: cont})
		if err != nil {
			return nil, err
		}

		for _, pod := range pods.Items {
			info := PodInfo{
				IsRunning: pod.Status.Phase == v1.PodRunning,
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Host:      pod.Status.HostIP,
				Owners:    make(map[string]string),
			}
			for _, owner := range pod.OwnerReferences {
				info.Owners[owner.Name] = owner.Kind
			}
			namespaces[pod.Namespace] = true

			var containers []ContainerInfo
			for _, container := range pod.Spec.Containers {
				containers = append(containers, ContainerInfo{
					Image: container.Image,
					Name:  container.Name,
				})
			}
			info.Containers = containers
			podList = append(podList, info)
		}

		cont = pods.Continue
		if cont == "" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return podList, nil
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

// IndexFrom creates a new cluster index for relevant cluster entities.
func IndexFrom(info *Info) *Index {
	index := &Index{
		Containers:   make(Set),
		DaemonSets:   make(Set),
		Deployments:  make(Set),
		Namespaces:   make(Set),
		Nodes:        make(Set),
		Pods:         make(Set),
		Running:      make(Set),
		StatefulSets: make(Set),
	}
	info.Apply(index)

	return index
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
		case "DaemonSet":
			index.DaemonSets.Add(n)
			break
		case "ReplicaSet": // hackish way to calculate deployments
			index.Deployments.Add(n)
			break
		case "StatefulSet":
			index.StatefulSets.Add(n)
		}
	}
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

// AgentSize takes the summary details and calculates the appropriate resource limits
// for the Instana agent.
func AgentSize(summary Summary) AgentLimits {
	size := summary.Deployments + summary.Namespaces

	if size > 2000 {
		return AgentLimits{
			CPULimit:      "2",
			CPURequest:    "500m",
			MemoryLimit:   "2Gi",
			MemoryRequest: "2Gi",
			Heap:          "800M",
		}
	} else if size > 1000 {
		return AgentLimits{
			CPULimit:      "2",
			CPURequest:    "500m",
			MemoryLimit:   "1Gi",
			MemoryRequest: "1Gi",
			Heap:          "400M",
		}
	}

	return AgentLimits{
		CPULimit:      "1.5",
		CPURequest:    "500m",
		MemoryLimit:   "512Mi",
		MemoryRequest: "512Mi",
		Heap:          "170M",
	}
}

// AgentLimits describes the perscribed limits for the Instana agent.
type AgentLimits struct {
	CPULimit      string
	CPURequest    string
	Heap          string
	MemoryLimit   string
	MemoryRequest string
}
