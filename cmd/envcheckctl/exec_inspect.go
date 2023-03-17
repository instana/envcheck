package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/instana/envcheck/agent"
	"github.com/instana/envcheck/cluster"
)

// ExecInspect executes the subcommand inspect.
func ExecInspect(config EnvcheckConfig) {
	log.SetFlags(0)
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

	log.Printf("pods=%d, running=%d, nodes=%d, containers=%d, namespaces=%d, deployments=%d, replicaSets=%d, daemonsets=%d, statefulsets=%d, duration=%v\n\n",
		summary.Pods,
		summary.Running,
		summary.Nodes,
		summary.Containers,
		summary.Namespaces,
		summary.Deployments,
		summary.Deployments,
		summary.DaemonSets,
		summary.StatefulSets,
		info.Finished.Sub(info.Started))
	log.Printf("coverage=\"%d of %d (%0.2f%%)\"\n\n", index.AgentRestarts.Len(), index.Nodes.Len(), float64(index.AgentRestarts.Len())/float64(index.Nodes.Len())*100.0)

	PrintKind(info.ServerVersion)
	PrintTop(10, "agentRestarts", index.AgentRestarts)
	PrintCounter("agentStatus", index.AgentStatus)
	PrintCounter("chartVersions", index.ChartVersions)
	PrintCounter("cniPlugins", index.CNIPlugins)
	PrintCounter("containerRuntimes", index.ContainerRuntimes)
	PrintCounter("instanceTypes", index.InstanceTypes)
	PrintCounter("kernels", index.KernelVersions)
	PrintCounter("kubelet", index.KubeletVersions)
	PrintCounter("osImages", index.OSImages)
	PrintCounter("podStatus", index.PodStatus)
	PrintCounter("proxy", index.ProxyVersions)
	PrintCounter("zones", index.Zones)
	PrintCounter("linkedConfigMaps", index.LinkedConfigMaps)

	size := agent.Size(summary)
	log.Printf("\nsizing=instana-agent cpurequests=%s cpulimits=%s memoryrequests=%s memorylimits=%s heap=%s\n",
		size.CPURequest,
		size.CPULimit,
		size.MemoryRequest,
		size.MemoryLimit,
		size.Heap)
}

func PrintKind(version string) {
	dist := ExtractDistribution(version)
	log.Println("serverDistribution:")
	log.Println(" -", dist)
	log.Println("")
	log.Println("serverVersion:")
	log.Println(" -", version)
	log.Println("")
}

func ExtractDistribution(version string) string {
	distribution := "kubernetes"
	if strings.Contains(version, "gke") {
		distribution = "openshift"
	} else if strings.Contains(version, "gke") {
		distribution = "gke"
	} else if strings.Contains(version, "eks") {
		distribution = "eks"
	}
	return distribution
}

type top struct {
	name  string
	value int
}

func PrintTop(n int, header string, c cluster.Counter) {
	var li []top
	for k, v := range c {
		li = append(li, top{k, v})
	}
	sort.Slice(li, func(i, j int) bool {
		return li[i].value > li[j].value
	})
	log.Println(header)
	if n > len(li) {
		n = len(li)
	}
	for _, v := range li[:n] {
		log.Printf("- \"%v\"=%d", v.name, v.value)
	}
}

func PrintCounter(header string, c cluster.Counter) {
	log.Println("")
	var keys []string
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	log.Println(header)
	for _, k := range keys {
		log.Printf("- \"%v\"=%d", k, c[k])
	}
	if len(keys) == 0 {
		log.Println(" - \"no known resource found\"")
	}
}

// QueryLive queries a cluster and builds the cluster info from the current data.
func QueryLive(query cluster.Query) (*cluster.Info, error) {
	info := &cluster.Info{
		Name:    query.Host(),
		Started: query.Time(),
	}

	log.Printf("envcheckctl=%s, cluster=%v, start=%v\n", Revision, info.Name, info.Started.Format(time.RFC3339))
	log.Println("Collecting cluster details. Duration varies depending on the cluster.")
	versionInfo, err := query.ServerVersion()
	if err != nil {
		return nil, err
	}
	info.ServerVersion = versionInfo

	pods, err := query.AllPods()
	if err != nil {
		return nil, err
	}
	info.Finished = query.Time()
	info.Pods = pods
	info.PodCount = len(pods)

	nodes, err := query.AllNodes()
	if err != nil {
		return nil, err
	}
	info.Nodes = nodes
	info.NodeCount = len(nodes)

	return info, nil
}

// LoadInfo loads cluster details from the specified reader.
func LoadInfo(r io.Reader) (*cluster.Info, error) {
	var info cluster.Info
	dec := json.NewDecoder(r)
	err := dec.Decode(&info)
	return &info, err
}
