package main

import (
	"expvar"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/instana/envcheck"
)

var (
	Revision = "dev"
)

func Exec(address string, info envcheck.DownwardInfo, c *http.Client) error {
	log.Printf("pinger=%s ping=%s pod=%s/%s podIP=%s nodeIP=%s", Revision, address, info.Namespace, info.Name, info.PodIP, info.NodeIP)
	publish("address", address)
	publish("name", info.Name)
	publish("namespace", info.Namespace)
	publish("nodeIP", info.NodeIP)
	publish("podIP", info.PodIP)

	client := envcheck.New(c)

	ping(client, address, info)

	return nil
}

func ping(client *envcheck.Client, address string, info envcheck.DownwardInfo) {
	success := false
	for true {
		err := client.Ping(address, info)
		time.Sleep(5 * time.Second)
		if err != nil {
			log.Printf("ping=failure pod=%s/%s address=%s err='%v'", info.Namespace, info.Name, address, err)
			success = false
			continue
		}

		if !success {
			log.Printf("ping=success pod=%s/%s address=%s\n", info.Namespace, info.Name, address)
		}
		success = true
	}
}

func main() {
	var host string
	var port string
	var downward envcheck.DownwardInfo

	flag.StringVar(&host, "address", os.Getenv("PINGHOST"), "the host to ping.")
	flag.StringVar(&port, "port", os.Getenv("PINGPORT"), "the port to ping.")
	flag.StringVar(&downward.Name, "name", os.Getenv("NAME"), "name of this pod.")
	flag.StringVar(&downward.Namespace, "namespace", os.Getenv("NAMESPACE"), "namespace this service is running in.")
	flag.StringVar(&downward.NodeIP, "nodeip", os.Getenv("NODEIP"), "node IP this service is running on.")
	flag.StringVar(&downward.PodIP, "podip", os.Getenv("PODIP"), "pod IP this service is running on.")

	flag.Parse()

	client := newClient()
	err := Exec(fmt.Sprintf("%s:%s", host, port), downward, client)
	if err != nil {
		log.Printf("status=shutdown error='%v'\n", err)
		os.Exit(1)
	}
}

func newClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   1 * time.Second,
				KeepAlive: 3 * time.Second,
			}).Dial,
			ResponseHeaderTimeout: 2 * time.Second,
		},
		Timeout: 6 * time.Second,
	}
}

func publish(key string, value string) {
	expvar.NewString(key).Set(value)
}
