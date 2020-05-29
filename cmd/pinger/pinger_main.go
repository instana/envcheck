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

	"github.com/instana/envcheck/ping"
	"github.com/jackpal/gateway"
)

var (
	// Revision is the Git commit SHA injected at compile time.
	Revision = "dev"
)

// Exec is the primary execution for the pinger application.
func Exec(address string, info ping.DownwardInfo, c *http.Client) error {
	gw, err := gateway.DiscoverGateway()
	if err != nil {
		log.Printf("discovergateway=failure pod=%s/%s err='%v'", info.Namespace, info.Name, err)
	}
	log.Printf("pinger=%s ping=%s pod=%s/%s podIP=%s nodeIP=%s gw=%v", Revision, address, info.Namespace, info.Name, info.PodIP, info.NodeIP, gw)
	publish("address", address)
	publish("name", info.Name)
	publish("namespace", info.Namespace)
	publish("nodeIP", info.NodeIP)
	publish("podIP", info.PodIP)

	client := ping.New(c)

	pingLoop(client, address, info)

	return nil
}

func pingLoop(client *ping.Client, address string, info ping.DownwardInfo) {
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
	var downward ping.DownwardInfo

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
		log.Printf("status=shutdown pod=%s/%s error='%v'\n", downward.Namespace, downward.Name, err)
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
