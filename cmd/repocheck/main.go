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

// Revision is the sha version for this application.
var Revision = "dev"

// Accumulator is used to capture a given time periods failure and total request count.
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
	podName := flag.String("name", os.Getenv("POD_NAME"), "pod name from downward API")
	tickRate := flag.Duration("tick", 1*time.Minute, "tick duration")
	shortReset := flag.Duration("short", 1*time.Hour, "short reset period")
	longReset := flag.Duration("long", 24*time.Hour, "long reset period")

	flag.Parse()
	log.SetFlags(log.LUTC | log.Lshortfile | log.LstdFlags)
	log.Printf("name=%s app=repocheck@%s key=%s tick=%v short=%v long=%v\n", *podName, Revision, *agentKey, *tickRate, *shortReset, *longReset)

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

	go resetAccumulator(shortTicker.C, *podName, &shortAccum, &lock)
	go resetAccumulator(longTicker.C, *podName, &longAccum, &lock)

	artifactURL := fmt.Sprintf("https://_:%s@artifact-public.instana.io/artifactory/features-public/com/instana/agent-feature/1.0.0-SNAPSHOT/agent-feature-1.0.0-20180125.135714-873-features.xml", *agentKey)
	ticker := time.NewTicker(*tickRate)

	for t := range ticker.C {
		var secondaryFail int
		var primaryFail int

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			resp, err := http.Get(*secondaryURL)
			if err != nil || resp.StatusCode != 200 {
				var code = -1
				if err == nil {
					code = resp.StatusCode
				}
				log.Printf("name=%s get=failed status=%d host=%s requested=%v err=`%v`\n", *podName, code, *secondaryURL, t, err)
				secondaryFail = 1
			}
			wg.Done()
		}()

		go func() {
			resp, err := http.Get(artifactURL)
			if err != nil || resp.StatusCode != 200 {
				var code = -1
				if err == nil {
					code = resp.StatusCode
				}
				log.Printf("name=%s get=failed status=%d host=artifact-public.instana.io requested=%v err=`%v`\n", *podName, code, t, err)
				primaryFail = 1
			}
			wg.Done()
		}()

		wg.Wait()

		lock.Lock()
		url := *secondaryURL
		v := shortAccum.Failures[url]
		shortAccum.Failures[url] = v + secondaryFail
		lv := longAccum.Failures[url]
		longAccum.Failures[url] = lv + secondaryFail

		url = "artifact-public.instana.io"
		v = shortAccum.Failures[url]
		shortAccum.Failures[url] = v + primaryFail
		lv = longAccum.Failures[url]
		longAccum.Failures[url] = lv + primaryFail

		shortAccum.Count++
		longAccum.Count++
		lock.Unlock()
	}
}

func resetAccumulator(ch <-chan time.Time, name string, data *Accumulator, lock *sync.Mutex) {
	for t := range ch {
		lock.Lock()
		for k, v := range data.Failures {
			var percentage float64 = 0.0
			if data.Count > 0 {
				percentage = float64(v) / float64(data.Count) * 100.0
			}
			log.Printf("name=%s period=%v failures=%v/%v(%v%%) host=%s end=%v \n", name, data.Period, v, data.Count, percentage, k, t)
			data.Failures[k] = 0
		}
		data.Count = 0
		lock.Unlock()
	}
}
