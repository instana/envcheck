package main

import (
	"expvar"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/instana/envcheck/network"
	"github.com/instana/envcheck/ping"
	"github.com/jackpal/gateway"
)

var (
	// Revision is the Git commit SHA injected at compile time.
	Revision = "dev"
)

// Exec is the primary execution for the pinger application.
func Exec(host string, port int, info ping.DownwardInfo, c *http.Client) error {
	gw, err := gateway.DiscoverGateway()
	if err != nil {
		log.Printf("pod=%s/%s discovergateway=failure err='%v'", info.Namespace, info.Name, err)
	}

	if host == "" {
		host = gw.String()
		log.Printf("pod=%s/%s host=gateway(%s)", info.Namespace, info.Name, host)
	}

	address := fmt.Sprintf("%s:%d", host, port)

	log.Printf("pod=%s/%s ping=%s podIP=%s nodeIP=%s gw=%v revision=%v", info.Namespace, info.Name, address, info.PodIP, info.NodeIP, gw, Revision)
	publish("address", address)
	publish("name", info.Name)
	publish("namespace", info.Namespace)
	publish("nodeIP", info.NodeIP)
	publish("podIP", info.PodIP)

	ifs, err := network.MapInterfaces()
	if err != nil {
		return err
	}

	for k, v := range ifs {
		log.Printf("pod=%s/%s if=%s ips=%v\n", info.Namespace, info.Name, k, v)
	}

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
			log.Printf("pod=%s/%s ping=failure address=%s err='%v'", info.Namespace, info.Name, address, err)
			success = false
			continue
		}

		if !success {
			log.Printf("pod=%s/%s ping=success address=%s\n", info.Namespace, info.Name, address)
		}
		success = true
	}
}

func main() {
	var host string
	var port int
	var downward ping.DownwardInfo
	var envPort = os.Getenv("PINGPORT")
	var defaultPort = 42700
	if envPort != "" {
		p, err := strconv.Atoi(envPort)
		if err != nil {
			log.Printf("Error unable to convert $PINGPORT to int: %v\n", err)
		} else {
			defaultPort = p
		}
	}

	flag.StringVar(&host, "address", os.Getenv("PINGHOST"), "the host to ping.")
	flag.IntVar(&port, "port", defaultPort, "the port to ping.")
	flag.StringVar(&downward.Name, "name", os.Getenv("NAME"), "name of this pod.")
	flag.StringVar(&downward.Namespace, "namespace", os.Getenv("NAMESPACE"), "namespace this service is running in.")
	flag.StringVar(&downward.NodeIP, "nodeip", os.Getenv("NODEIP"), "node IP this service is running on.")
	flag.StringVar(&downward.PodIP, "podip", os.Getenv("PODIP"), "pod IP this service is running on.")

	flag.Parse()

	client := newClient()
	err := Exec(host, port, downward, client)
	if err != nil {
		log.Printf("pod=%s/%s status=shutdown error='%v'\n", downward.Namespace, downward.Name, err)
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
