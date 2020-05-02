package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/instana/envcheck/cluster"
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

	info := &cluster.Info{
		Name: config.Host,
	}

	start := time.Now()
	log.Printf("envcheckctl=%s, cluster=%v, start=%v\n", Revision, info.Name, start.Format(time.RFC3339))
	log.Println("Collecting pod details. Duration varies depending on the cluster.")
	pods, err := cluster.AllPods(clientset)
	if err != nil {
		log.Fatalf("error retrieving pods: %v\n", err)
	}
	info.Pods = pods

	info.PodCount = len(pods)

	index := cluster.New()
	info.Apply(index)
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

	size := cluster.AgentSize(summary)
	log.Printf("sizing=instana-agent cpurequests=%s cpulimits=%s memoryrequests=%s memorylimits=%s heap=%s\n",
		size.CPURequest,
		size.CPULimit,
		size.MemoryRequest,
		size.MemoryLimit,
		size.Heap)

	w, err := os.Create(fmt.Sprintf("cluster-info-%d.json", time.Now().UTC().Unix()))
	if err != nil {
		log.Fatalln(err)
	}
	defer w.Close()

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	err = enc.Encode(info)
	if err != nil {
		log.Fatalln(err)
	}
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
