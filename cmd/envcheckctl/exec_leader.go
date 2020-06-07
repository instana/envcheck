package main

import (
	"fmt"
	"log"

	"github.com/instana/envcheck/cluster"
)

// ExecLeader executes the leader subcommand.
func ExecLeader(config EnvcheckConfig) {
	query, err := cluster.New(config.Kubeconfig)
	if err != nil {
		log.Fatalf("error initialising cluster query: %v\n", err)
	}

	leader, err := query.InstanaLeader()
	if err != nil {
		log.Fatalf("error retrieving leader: %v", err)
	}

	fmt.Println(leader)
}
