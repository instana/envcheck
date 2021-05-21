package main

import (
	"fmt"
	"github.com/instana/envcheck/cluster"
	"log"
)

// ExecAgent executes the agent debug sub-command.
func ExecAgent(config EnvcheckConfig) {
	query, err := cluster.New(config.Kubeconfig)
	if err != nil {
		log.Fatalf("error initialising cluster query: %v\n", err)
	}

	info, err := query.AgentInfo(config.AgentNamespace, config.AgentName)
	if err != nil {
		return
	}

	for _, v := range info.EventList {
		fmt.Printf("reason=%v message=`%v`\n", v.Reason, v.Message)
	}

	log.Printf("desired=%d ready=%d unavailable=%d misscheduled=%d\n", info.Desired, info.Ready, info.Unavailable, info.Misscheduled)
}
