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
	Revision string
)

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

type Set map[string]bool

func (s Set) Add(item string) {
	s[item] = true
}

func (s Set) Len() int {
	return len(s)
}

func (s Set) Contains(item string) bool {
	_, present := s[item]
	return present
}

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

func (index *ClusterIndex) Summary() ClusterSummary {
	return ClusterSummary{
		Containers:   len(index.Containers),
		DaemonSets:   len(index.DaemonSets),
		Deployments:  len(index.Deployments),
		Nodes:        len(index.Nodes),
		Namespaces:   len(index.Namespaces),
		Pods:         len(index.Pods),
		Running:      len(index.Running),
		StatefulSets: len(index.StatefulSets),
	}
}

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

type ClusterInfo struct {
	Name     string
	PodCount int
	Pods     []PodInfo
	Version  string
}

type PodApplyable interface {
	Each(PodInfo)
}

func (info *ClusterInfo) Apply(applyable ...PodApplyable) {
	for _, pod := range info.Pods {
		for _, a := range applyable {
			a.Each(pod)
		}
	}
}

type PodInfo struct {
	Containers []ContainerInfo
	Host       string
	IsRunning  bool
	Name       string
	Namespace  string
	Owners     map[string]string
}

type ContainerInfo struct {
	Name  string
	Image string
}
