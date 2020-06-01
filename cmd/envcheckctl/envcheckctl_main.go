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
				Host:      config.PingerHost,
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
	PingerHost      string
	PingerNamespace string
	Podfile         string
}

var (
	ErrNoSubcommand      = fmt.Errorf("no sub-command specified")
	ErrInvalidSubcommand = fmt.Errorf("invalid sub-command specified")
)

func Parse(args []string, kubepath string, w io.Writer) (*EnvcheckConfig, error) {
	var fs []*flag.FlagSet

	var daemonConfig = EnvcheckConfig{ApplyDaemon: true}
	daemonFlags := flag.NewFlagSet("daemon", flag.ExitOnError)
	daemonFlags.StringVar(&daemonConfig.AgentNamespace, "ns", "instana-agent", "daemon namespace")
	daemonFlags.StringVar(&daemonConfig.Kubeconfig, "kubeconfig", kubepath, "absolute path to the kubeconfig file")
	daemonFlags.SetOutput(w)
	fs = append(fs, daemonFlags)

	var inspectConfig = EnvcheckConfig{}
	inspectFlags := flag.NewFlagSet("inspect", flag.ExitOnError)
	inspectFlags.BoolVar(&inspectConfig.IsLive, "live", true, "retrieve pods from cluster")
	inspectFlags.StringVar(&inspectConfig.AgentNamespace, "agentns", "instana-agent", "Instana agent namespace")
	inspectFlags.StringVar(&inspectConfig.Podfile, "podfile", "", "podfile")
	inspectFlags.StringVar(&inspectConfig.Kubeconfig, "kubeconfig", kubepath, "absolute path to the kubeconfig file")
	inspectFlags.SetOutput(w)
	fs = append(fs, inspectFlags)

	var pingConfig = EnvcheckConfig{ApplyPinger: true}
	pingFlags := flag.NewFlagSet("ping", flag.ExitOnError)
	pingFlags.StringVar(&pingConfig.PingerHost, "host", "", "override IP or DNS name to ping. defaults to nodeIP if blank")
	pingFlags.StringVar(&pingConfig.PingerNamespace, "ns", "default", "ping client namespace")
	pingFlags.StringVar(&pingConfig.Kubeconfig, "kubeconfig", kubepath, "absolute path to the kubeconfig file")
	pingFlags.SetOutput(w)
	fs = append(fs, pingFlags)

	if len(args) < 2 {
		w.Write([]byte("Usage: " + args[0] + " requires a subcommand\n"))
		for _, v := range fs {
			v.Usage()
		}
		return nil, ErrNoSubcommand
	}

	cmdArgs := args[2:]
	switch args[1] {
	case "daemon":
		daemonFlags.Parse(cmdArgs)
		return &daemonConfig, nil
	case "inspect":
		inspectFlags.Parse(cmdArgs)
		return &inspectConfig, nil
	case "ping":
		pingFlags.Parse(cmdArgs)
		return &pingConfig, nil
	}

	w.Write([]byte("Usage: " + args[0] + " requires a subcommand\n"))
	for _, v := range fs {
		w.Write([]byte("\n"))
		v.Usage()
	}
	return nil, ErrInvalidSubcommand
}

func main() {
	kubepath := configPath()
	config, err := Parse(os.Args, kubepath, os.Stderr)
	if err != nil {
		os.Exit(1)
	}

	Exec(*config)
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
