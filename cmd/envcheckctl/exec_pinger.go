package main

import (
	"github.com/instana/envcheck/cluster"
	"log"
)

// ExecPinger executes the pinger subcommand.
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
