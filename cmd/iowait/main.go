package main

import (
	"github.com/instana/iowait/procfs"
	"log"
	"strconv"
	"time"
)

var Version = "dev"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.LUTC)
	log.Printf("app=iowait@%v ", Version)
	ticker := time.Tick(5 * time.Second)
	for t := range ticker {
		log.Printf("==== %v ================================================================\n", t)
		stats, err := procfs.ReadStats("/")
		if err != nil {
			log.Printf("err=`%v`", err)
			continue
		}

		var max = 0
		for _, s := range stats {
			if len(s.Name) > max {
				max = len(s.Name)
			}
		}
		max++

		format := "%" + strconv.Itoa(max) + "s -> %v\n"
		for _, s := range stats {
			log.Printf(format, s.Name, s.IOWait)
			if s.IOWait <= 0 {
				break
			}
		}
	}
}
