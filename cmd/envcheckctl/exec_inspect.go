package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/instana/envcheck/agent"
	"github.com/instana/envcheck/cluster"
)

// ExecInspect executes the inspect subcommand.
func ExecInspect(config EnvcheckConfig) {
	var info *cluster.Info
	if config.IsLive() {
		query, err := cluster.New(config.Kubeconfig)
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
	} else if config.Podfile != "" {
		r, err := os.Open(config.Podfile)
		if err != nil {
			log.Fatalf("open=failed file=%s err='%v'\n", config.Podfile, err)
		}
		info, err = LoadInfo(r)
		r.Close()
		if err != nil {
			log.Fatalf("read=failed file=%s err='%v'\n", config.Podfile, err)
		}
		log.Printf("envcheckctl=%s, cluster=%v, podfile=%v\n", Revision, info.Name, config.Podfile)
	}

	index := cluster.NewIndex()
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
		info.Finished.Sub(info.Started))

	size := agent.Size(summary)
	log.Printf("sizing=instana-agent cpurequests=%s cpulimits=%s memoryrequests=%s memorylimits=%s heap=%s\n",
		size.CPURequest,
		size.CPULimit,
		size.MemoryRequest,
		size.MemoryLimit,
		size.Heap)
}

// QueryLive queries a cluster and builds the cluster info from the current data.
func QueryLive(query cluster.Query) (*cluster.Info, error) {
	info := &cluster.Info{
		Name:    query.Host(),
		Started: query.Time(),
	}

	log.Printf("envcheckctl=%s, cluster=%v, start=%v\n", Revision, info.Name, info.Started.Format(time.RFC3339))
	log.Println("Collecting pod details. Duration varies depending on the cluster.")
	pods, err := query.AllPods()
	if err != nil {
		return nil, err
	}
	info.Finished = query.Time()
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
