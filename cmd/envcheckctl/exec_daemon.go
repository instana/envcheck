package main

import (
	"github.com/instana/envcheck/cluster"
	"log"
)

// ExecDaemon executes the daemon pinger subcommand.
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
