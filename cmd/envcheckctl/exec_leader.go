package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

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
		log.Fatalf("error retrieving leader: %v\n", err)
	}

	fmt.Println(leader)

	if config.Profile {
		err := DownloadFile(defaultProfilerURL, defaultFilename)
		if err != nil {
			log.Fatalf("error downloading profiler: %v\n", err)
		}
		// Upload and unpack file on pod
		// Exec memory profiler in pod
		// /tmp/profiler.sh -d ${DURATION} -e alloc -o collapsed -f alloc_profile_${agent_pid} \
		//      --title Instana_Agent_Memory_Allocation_Profile --minwidth 1 -t ${agent_pid}
		// Exec cpu profiler in pod
		// /tmp/profiler.sh -d ${DURATION} -e cpu -o collapsed -f cpu_profile_${agent_pid} \
		//      --title Instana_Agent_CPU_Profile --minwidth 1 -t -I 'com/instana/*' -X start_thread ${agent_pid}
		// Exec jmap in pod
		// jvm/bin/jmap -dump:format=b,file=heap_${agent_pid}.hprof ${agent_pid}
		// Download the profile
	}
}

const (
	defaultProfilerURL = "https://github.com/jvm-profiling-tools/async-profiler/releases/download/v1.7.1/async-profiler-1.7.1-linux-x64.tar.gz"
	defaultFilename    = "async-profiler.tgz"
)

// DownloadFile downloads the file at url to the filename.
func DownloadFile(url, filename string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, resp.Body)
	return err
}
