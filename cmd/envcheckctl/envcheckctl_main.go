package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// Revision is the Git commit SHA injected at compile time.
	Revision string
)

// Exec is the primary execution for the envcheckctl application.
func Exec(kubeconfig string) {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalln(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err.Error())
	}

	cluster := &ClusterInfo{
		Name: config.Host,
	}

	start := time.Now()
	log.Printf("envcheckctl=%s, cluster=%v, start=%v\n", Revision, cluster.Name, start.Format(time.RFC3339))
	log.Println("Collecting pod details. Duration varies depending on the cluster.")
	pods, err := AllPods(clientset)
	if err != nil {
		log.Fatalf("error retrieving pods: %v\n", err)
	}
	cluster.Pods = pods

	cluster.PodCount = len(pods)

	index := New()
	cluster.Apply(index)
	summary := index.Summary()

	log.Printf("pods=%d, running=%d, nodes=%d, containers=%d, namespaces=%d, deployments=%d, daemonsets=%d, statefulsets=%d, duration=%v\n",
		summary.Pods,
		summary.Running,
		summary.Nodes,
		summary.Containers,
		summary.Namespaces,
		summary.Deployments,
		summary.DaemonSets,
		summary.StatefulSets,
		time.Now().Sub(start))

	w, err := os.Create(fmt.Sprintf("cluster-info-%d.json", time.Now().UTC().Unix()))
	if err != nil {
		log.Fatalln(err)
	}
	defer w.Close()

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	err = enc.Encode(cluster)
	if err != nil {
		log.Fatalln(err)
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

// New creates a new cluster index for relevant cluster entities.
func New() *ClusterIndex {
	return &ClusterIndex{
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

// ClusterIndex provides indexes for a number of the cluster entities.
type ClusterIndex struct {
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
func (index *ClusterIndex) Summary() ClusterSummary {
	return ClusterSummary{
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
func (index *ClusterIndex) Each(pod PodInfo) {
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

// ClusterSummary provides a summary overview of the number of entities in the cluster.
type ClusterSummary struct {
	Containers   int
	DaemonSets   int
	Deployments  int
	Namespaces   int
	Nodes        int
	Pods         int
	Running      int
	StatefulSets int
}

// AllPods retrieves all pod info from the cluster.
func AllPods(clientset *kubernetes.Clientset) ([]PodInfo, error) {
	var cont string
	var podList []PodInfo
	namespaces := make(map[string]bool)

	for true {
		pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{Limit: 100, Continue: cont})
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

func main() {
	var kubeconfig string

	if home := homeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	Exec(kubeconfig)
}

func homeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}

// ClusterInfo is a data structure for relevant cluster data.
type ClusterInfo struct {
	Name     string
	PodCount int
	Pods     []PodInfo
	Version  string
}

// PodApplyable is the interface to receive pod info from a pod collection.
type PodApplyable interface {
	Each(PodInfo)
}

// Apply iterates over each pod and yields it to the list of applyables.
func (info *ClusterInfo) Apply(applyable ...PodApplyable) {
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
