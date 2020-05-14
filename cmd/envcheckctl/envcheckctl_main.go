package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/instana/envcheck/agent"
	"github.com/instana/envcheck/cluster"
)

var (
	// Revision is the Git commit SHA injected at compile time.
	Revision string
)

// QueryLive queries a cluster for the latest data.
func QueryLive(query cluster.Query) (*cluster.Info, error) {
	info := &cluster.Info{
		Name:    query.Host(),
		Started: time.Now(),
	}

	log.Printf("envcheckctl=%s, cluster=%v, start=%v\n", Revision, info.Name, info.Started.Format(time.RFC3339))
	log.Println("Collecting pod details. Duration varies depending on the cluster.")
	pods, err := query.AllPods()
	if err != nil {
		return nil, err
	}
	info.Finished = time.Now()
	info.Pods = pods
	info.PodCount = len(pods)

	return info, nil
}

// LoadInfo loads cluster details from the specified reader.
func LoadInfo(r io.Reader) (*cluster.Info, error) {
	var info cluster.Info
	dec := json.NewDecoder(r)
	err := dec.Decode(&info)
	return &info, err
}

// Exec is the primary execution for the envcheckctl application.
func Exec(kubeconfig string, isLive bool, podfile string) {
	var info *cluster.Info

	// if podfile is set, disable live query
	isLive = podfile == ""

	if isLive {
		query, err := cluster.New(kubeconfig)
		if err != nil {
			log.Fatalf("error initialising cluster query: %v\n", err)
		}

		info, err = QueryLive(query)
		if err != nil {
			log.Fatalf("error retrieving cluster info: %v\n", err)
		}

		filename := fmt.Sprintf("cluster-info-%d.json", time.Now().UTC().Unix())
		w, err := os.Create(filename)
		if err != nil {
			log.Fatalln(err)
		}

		enc := json.NewEncoder(w)
		err = enc.Encode(info)
		w.Close()
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("podfile=%s", filename)
	} else if podfile != "" {
		r, err := os.Open(podfile)
		info, err = LoadInfo(r)
		r.Close()
		if err != nil {
			log.Fatalf("error loading cluster info: %v\n", err)
		}
		log.Printf("envcheckctl=%s, cluster=%v, podfile=%v\n", Revision, info.Name, podfile)
	} else {
		fmt.Println("Either a podfile must be provided or live must be set to true")
		flag.Usage()
		os.Exit(1)
	}

	index := cluster.IndexFrom(info)
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
		info.Finished.Sub(info.Started))

	size := agent.Size(summary)
	log.Printf("sizing=instana-agent cpurequests=%s cpulimits=%s memoryrequests=%s memorylimits=%s heap=%s\n",
		size.CPURequest,
		size.CPULimit,
		size.MemoryRequest,
		size.MemoryLimit,
		size.Heap)
}

func main() {
	var kubeconfig string
	var podfile string
	var isLive bool

	flag.BoolVar(&isLive, "live", true, "retrieve pods from cluster")
	flag.StringVar(&podfile, "podfile", "", "podfile")
	flag.StringVar(&kubeconfig, "kubeconfig", configPath(), "absolute path to the kubeconfig file")
	flag.Parse()

	Exec(kubeconfig, isLive, podfile)
}

func configPath() string {
	usr, err := user.Current()
	if err != nil {
		log.Println(err)
		return ""
	}
	return filepath.Join(usr.HomeDir, ".kube", "config")
}
