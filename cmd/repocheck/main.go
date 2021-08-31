package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// Version is the sha version for this application.
var Version = "dev"

type Accumulator struct {
	Failures map[string]int
	Count    int
	Period   time.Duration
}

func main() {
	var lock sync.Mutex
	var shortAccum = Accumulator{Failures: make(map[string]int)}
	var longAccum = Accumulator{Failures: make(map[string]int)}

	secondaryURL := flag.String("secondary", "https://www.google.com", "secondary check url")
	agentKey := flag.String("key", os.Getenv("INSTANA_AGENT_KEY"), "instana agent key")
	tickRate := flag.Duration("tick", 1*time.Minute, "tick duration")
	shortReset := flag.Duration("short", 1*time.Hour, "short reset period")
	longReset := flag.Duration("long", 24*time.Hour, "long reset period")

	flag.Parse()
	log.SetFlags(log.LUTC | log.Lshortfile)
	log.Printf("app=repocheck@%s key=%s tick=%v short=%v long=%v\n", Version, *agentKey, *tickRate, *shortReset, *longReset)

	if *agentKey == "" {
		log.Fatalln("err=`agent key is required, none specified`")
	}

	if *secondaryURL == "" {
		log.Fatalln("err=`secondary URL is required, none specified`")
	}

	shortAccum.Period = *shortReset
	shortTicker := time.NewTicker(*shortReset)
	longAccum.Period = *longReset
	longTicker := time.NewTicker(*longReset)

	url := "artifact-public.instana.io"
	shortAccum.Failures[*secondaryURL] = 0
	shortAccum.Failures[url] = 0
	longAccum.Failures[*secondaryURL] = 0
	longAccum.Failures[url] = 0

	go resetAccumulator(shortTicker.C, &shortAccum, &lock)
	go resetAccumulator(longTicker.C, &longAccum, &lock)

	artifactURL := fmt.Sprintf("https://_:%s@artifact-public.instana.io/artifactory/features-public/com/instana/agent-feature/1.0.0-SNAPSHOT/agent-feature-1.0.0-20180125.135714-873-features.xml", *agentKey)
	ticker := time.NewTicker(*tickRate)

	for t := range ticker.C {
		lock.Lock()
		shortAccum.Count++
		longAccum.Count++
		resp, err := http.Get(*secondaryURL)
		if err != nil || resp.StatusCode != 200 {
			log.Printf("get=failed host=%s requested=%v err=`%v`\n", *secondaryURL, t, err)
			url := *secondaryURL
			v := shortAccum.Failures[url]
			shortAccum.Failures[url] = v + 1
			longAccum.Failures[url] = v + 1
		}

		resp, err = http.Get(artifactURL)
		if err != nil || resp.StatusCode != 200 {
			var code = -1
			if err == nil {
				code = resp.StatusCode
			}
			log.Printf("get=failed status=%d host=artifact-public.instana.io requested=%v err=`%v`\n", code, t, err)
			url := "artifact-public.instana.io"
			v := shortAccum.Failures[url]
			shortAccum.Failures[url] = v + 1
			longAccum.Failures[url] = v + 1
		}
		lock.Unlock()
	}
}

func resetAccumulator(ch <-chan time.Time, data *Accumulator, lock *sync.Mutex) {
	for t := range ch {
		lock.Lock()
		for k, v := range data.Failures {
			log.Printf("host=%s failures=%v/%v(%v%%) end=%v period=%v\n", k, v, data.Count, (v / data.Count * 100.0), t, data.Period)
			data.Failures[k] = 0
		}
		data.Count = 0
		lock.Unlock()
	}
}
