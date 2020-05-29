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
func Exec(config EnvcheckConfig) {
	var info *cluster.Info

	if config.ApplyDaemon || config.ApplyPinger {
		command, err := cluster.NewCommand(config.Kubeconfig)
		if err != nil {
			log.Fatalf("error initialising cluster command: %v\n", err)
		}

		if config.ApplyDaemon {
			dc := cluster.DaemonConfig{
				Image:     "instana/envcheck-daemon:latest",
				Namespace: config.AgentNamespace,
				Host:      "0.0.0.0",
				Port:      42700,
				Version:   Revision,
			}
			err := command.CreateDaemon(dc)
			if err != nil {
				log.Fatalf("error creating daemon: %v\n", err)
			}
		}

		if config.ApplyPinger {
			pc := cluster.PingerConfig{
				Image:     "instana/envcheck-pinger:latest",
				Namespace: config.PingerNamespace,
				Version:   Revision,
				Port:      42700,
			}
			err := command.CreatePinger(pc)
			if err != nil {
				log.Fatalf("error creating ping client: %v\n", err)
			}
		}
		return
	}

	// if podfile is set, disable live query
	config.IsLive = config.Podfile == ""

	if config.IsLive {
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
		info, err = LoadInfo(r)
		r.Close()
		if err != nil {
			log.Fatalf("error loading cluster info: %v\n", err)
		}
		log.Printf("envcheckctl=%s, cluster=%v, podfile=%v\n", Revision, info.Name, config.Podfile)
	} else {
		fmt.Println("Either a podfile must be provided or live must be set to true")
		flag.Usage()
		os.Exit(1)
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

type EnvcheckConfig struct {
	AgentNamespace  string
	ApplyDaemon     bool
	ApplyPinger     bool
	IsLive          bool
	Kubeconfig      string
	PingerNamespace string
	Podfile         string
}

func main() {
	var config EnvcheckConfig

	flag.StringVar(&config.AgentNamespace, "agentns", "instana-agent", "Instana agent namespace")
	flag.StringVar(&config.Kubeconfig, "kubeconfig", configPath(), "absolute path to the kubeconfig file")
	flag.BoolVar(&config.ApplyDaemon, "daemon", false, "deploy daemon to cluster")
	flag.BoolVar(&config.ApplyPinger, "ping", false, "deploy ping client to cluster")
	flag.BoolVar(&config.IsLive, "live", true, "retrieve pods from cluster")
	flag.StringVar(&config.PingerNamespace, "pingns", "default", "ping client namespace")
	flag.StringVar(&config.Podfile, "podfile", "", "podfile")
	flag.Parse()

	Exec(config)
}

func configPath() string {
	env := os.Getenv("KUBECONFIG")
	if env != "" {
		return env
	}
	usr, err := user.Current()
	if err != nil {
		log.Println(err)
		return ""
	}
	return filepath.Join(usr.HomeDir, ".kube", "config")
}
