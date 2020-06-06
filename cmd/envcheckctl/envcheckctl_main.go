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
	"runtime"
	"time"

	"github.com/instana/envcheck/agent"
	"github.com/instana/envcheck/cluster"
)

var (
	// Revision is the Git commit SHA injected at compile time.
	Revision string = "dev"
)

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

func ExecDaemon(config EnvcheckConfig) {
	command, err := cluster.NewCommand(config.Kubeconfig)
	if err != nil {
		log.Fatalf("createClient=failed err='%v'\n", err)
	}

	dc := cluster.DaemonConfig{
		Image:     "instana/envcheck-daemon:latest",
		Namespace: config.AgentNamespace,
		Host:      "0.0.0.0",
		Port:      42700,
		Version:   Revision,
	}
	err = command.CreateDaemon(dc)
	if err != nil {
		log.Fatalf("createDaemon=failed err='%v'\n", err)
	}
}

func ExecPinger(config EnvcheckConfig) {
	command, err := cluster.NewCommand(config.Kubeconfig)
	if err != nil {
		log.Fatalf("createClient=failed err='%v'\n", err)
	}

	pc := cluster.PingerConfig{
		Image:      "instana/envcheck-pinger:latest",
		Namespace:  config.PingerNamespace,
		Version:    Revision,
		Host:       config.PingerHost,
		Port:       42700,
		UseGateway: config.UseGateway,
	}
	err = command.CreatePinger(pc)
	if err != nil {
		log.Fatalf("createPinger=failed err='%v'\n", err)
	}
}

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

func ExecVersion(w io.Writer) {
	w.Write([]byte(fmt.Sprintf("revision=%s go=%s\n", Revision, runtime.Version())))
}

// Exec is the primary execution for the envcheckctl application.
func Exec(config EnvcheckConfig) {
	switch config.Subcommand {
	case ApplyDaemon:
		ExecDaemon(config)
	case ApplyPinger:
		ExecPinger(config)
	case InspectCluster:
		ExecInspect(config)
	case PrintVersion:
		ExecVersion(os.Stdout)
	}
}

// EnvcheckConfig is the primary configuration parameters that control this exe.
type EnvcheckConfig struct {
	AgentNamespace  string
	Kubeconfig      string
	PingerHost      string
	PingerNamespace string
	Podfile         string
	Subcommand      int
	UseGateway      bool
}

func (c *EnvcheckConfig) IsLive() bool {
	return c.Podfile == ""
}

var (
	// ErrNoSubcommand occurs when too few arguments are supplied to the executable.
	ErrNoSubcommand = fmt.Errorf("no sub-command specified")
	// ErrUnknownSubcommand occurs when an unknown sub-command is specified.
	ErrUnknownSubcommand = fmt.Errorf("invalid sub-command specified")
)

const (
	ApplyDaemon int = iota
	ApplyPinger
	InspectCluster
	PrintVersion
)

// Parse parses the individual subcommands and returns the related configuration.
func Parse(args []string, kubepath string, w io.Writer) (*EnvcheckConfig, error) {
	var fs []*flag.FlagSet

	var daemonConfig = EnvcheckConfig{Subcommand: ApplyDaemon}
	daemonFlags := flag.NewFlagSet("daemon", flag.ExitOnError)
	daemonFlags.StringVar(&daemonConfig.AgentNamespace, "ns", "instana-agent", "daemon namespace")
	daemonFlags.StringVar(&daemonConfig.Kubeconfig, "kubeconfig", kubepath, "absolute path to the kubeconfig file")
	daemonFlags.SetOutput(w)
	fs = append(fs, daemonFlags)

	var inspectConfig = EnvcheckConfig{Subcommand: InspectCluster}
	inspectFlags := flag.NewFlagSet("inspect", flag.ExitOnError)
	inspectFlags.StringVar(&inspectConfig.AgentNamespace, "agentns", "instana-agent", "Instana agent namespace")
	inspectFlags.StringVar(&inspectConfig.Podfile, "podfile", "", "read from podfile instead of live cluster query")
	inspectFlags.StringVar(&inspectConfig.Kubeconfig, "kubeconfig", kubepath, "absolute path to the kubeconfig file")
	inspectFlags.SetOutput(w)
	fs = append(fs, inspectFlags)

	var pingConfig = EnvcheckConfig{Subcommand: ApplyPinger}
	pingFlags := flag.NewFlagSet("ping", flag.ExitOnError)
	pingFlags.StringVar(&pingConfig.PingerHost, "host", "", "override IP or DNS name to ping. defaults to nodeIP if blank")
	pingFlags.StringVar(&pingConfig.PingerNamespace, "ns", "default", "ping client namespace")
	pingFlags.StringVar(&pingConfig.Kubeconfig, "kubeconfig", kubepath, "absolute path to the kubeconfig file")
	pingFlags.BoolVar(&pingConfig.UseGateway, "use-gateway", false, "use the pods gateway as the host to ping")
	pingFlags.SetOutput(w)
	fs = append(fs, pingFlags)

	versionFlags := flag.NewFlagSet("version", flag.ExitOnError)
	fs = append(fs, versionFlags)

	if len(args) < 2 {
		usage(args[0], w, fs)
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
	case "version":
		return &EnvcheckConfig{Subcommand: PrintVersion}, nil
	}

	usage(args[0], w, fs)
	return nil, ErrUnknownSubcommand
}

func usage(cmd string, w io.Writer, fs []*flag.FlagSet) {
	w.Write([]byte("Usage: " + cmd + " requires a subcommand (rev. " + Revision + ")\n"))
	for _, v := range fs {
		w.Write([]byte("\n"))
		v.Usage()
	}
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
