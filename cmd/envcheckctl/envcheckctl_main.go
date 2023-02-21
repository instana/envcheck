package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

var (
	// Revision is the Git commit SHA injected at compile time.
	Revision = "dev"
)

// ExecVersion prints the current revision and go runtime version to the writer.
func ExecVersion(w io.Writer) {
	w.Write([]byte(fmt.Sprintf("revision=%s go=%s\n", Revision, runtime.Version())))
}

// Exec is the primary execution for the envcheckctl application.
func Exec(config EnvcheckConfig) {
	switch config.Subcommand {
	case Agent:
		ExecAgent(config)
	case ApplyDaemon:
		ExecDaemon(config)
	case ApplyPinger:
		ExecPinger(config)
	case InspectCluster:
		ExecInspect(config)
	case Leader:
		ExecLeader(config)
	case PrintVersion:
		ExecVersion(os.Stdout)
	}
}

// EnvcheckConfig is the primary configuration parameters that control this exe.
type EnvcheckConfig struct {
	AgentNamespace  string
	AgentName       string
	Kubeconfig      string
	PingerHost      string
	PingerNamespace string
	Podfile         string
	Profile         bool
	Subcommand      int
	UseGateway      bool
}

// IsLive indicates whether the inspect details should be loaded from an API or file.
func (c *EnvcheckConfig) IsLive() bool {
	return c.Podfile == ""
}

const (
	// Agent is the subcommand flag to indicate the agent debug to be executed.
	Agent int = iota
	// ApplyDaemon is the subcommand flag to indicate the daemon to be executed.
	ApplyDaemon
	// ApplyPinger is the subcommand flag to indicate the pinger to be executed.
	ApplyPinger
	// InspectCluster is the subcommand flag to indicate the inspect to be executed.
	InspectCluster
	// Leader is the subcommand enum to indicate leader commands should be executed.
	Leader
	// PrintVersion is the subcommand flag to indicate the version print to be executed.
	PrintVersion
)

// Parse parses the individual subcommands and returns the related configuration.
func Parse(args []string, kubepath string, w io.Writer) (*EnvcheckConfig, error) {
	cmdFlags := New(w)

	flags, config := cmdFlags.FlagSet("agent", Agent)
	flags.StringVar(&config.AgentNamespace, "ns", "instana-agent", "agent namespace")
	flags.StringVar(&config.AgentName, "name", "instana-agent", "agent daemonset name")
	flags.StringVar(&config.Kubeconfig, "kubeconfig", kubepath, "absolute path to the kubeconfig file")

	flags, config = cmdFlags.FlagSet("daemon", ApplyDaemon)
	flags.StringVar(&config.AgentNamespace, "ns", "instana-agent", "daemon namespace")
	flags.StringVar(&config.Kubeconfig, "kubeconfig", kubepath, "absolute path to the kubeconfig file")

	flags, config = cmdFlags.FlagSet("ping", ApplyPinger)
	flags.StringVar(&config.PingerHost, "host", "", "override IP or DNS name to ping. defaults to nodeIP if blank")
	flags.StringVar(&config.PingerNamespace, "ns", "default", "ping client namespace")
	flags.StringVar(&config.Kubeconfig, "kubeconfig", kubepath, "absolute path to the kubeconfig file")
	flags.BoolVar(&config.UseGateway, "use-gateway", false, "use the pods gateway as the host to ping")

	flags, config = cmdFlags.FlagSet("inspect", InspectCluster)
	flags.StringVar(&config.Podfile, "podfile", "", "read from podfile instead of live cluster query")
	flags.StringVar(&config.Kubeconfig, "kubeconfig", kubepath, "absolute path to the kubeconfig file")

	flags, config = cmdFlags.FlagSet("leader", Leader)
	flags.StringVar(&config.Kubeconfig, "kubeconfig", kubepath, "absolute path to the kubeconfig file")
	flags.BoolVar(&config.Profile, "profile", false, "attach a profiler to the agent")

	cmdFlags.FlagSet("version", PrintVersion)

	return cmdFlags.Parse(args)
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
