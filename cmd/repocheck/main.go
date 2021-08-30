package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// Version is the sha version for this application.
var Version = "dev"

func main() {
	agentKey := flag.String("key", os.Getenv("INSTANA_AGENT_KEY"), "instana agent key")
	tickRate := flag.Duration("tick", 10*time.Minute, "tick duration")
	flag.Parse()
	log.SetFlags(log.LUTC | log.Lshortfile)
	log.Printf("app=repocheck@%s key=%s\n", Version, *agentKey)

	if *agentKey == "" {
		log.Fatalln("err=`agent key is required, none specified`")
	}

	artifactURL := fmt.Sprintf("https://_:%s@artifact-public.instana.io/artifactory/features-public/com/instana/agent-feature/1.0.0-SNAPSHOT/agent-feature-1.0.0-20180125.135714-873-features.xml", *agentKey)
	ticker := time.NewTicker(*tickRate)
	for t := range ticker.C {
		resp, err := http.Get("https://www.google.com")
		if err != nil || resp.StatusCode != 200 {
			log.Printf("get=failed host=www.google.com requested=%v err=`%v`\n", t, err)
		}

		resp, err = http.Get(artifactURL)
		if err != nil || resp.StatusCode != 200 {
			log.Printf("get=failed status=%d host=artifact-public.instana.io requested=%v err=`%v`\n", resp.StatusCode, t, err)
		}
	}
}
