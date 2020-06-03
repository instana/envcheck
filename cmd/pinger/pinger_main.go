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
	log.SetFlags(log.LUTC | log.Lmsgprefix | log.LstdFlags)
	log.SetPrefix(fmt.Sprintf("pod=%s/%s ", info.Namespace, info.Name))

	gw, err := gateway.DiscoverGateway()
	if err != nil {
		log.Printf("discovergateway=failure err='%v'", err)
	}

	if host == "" {
		host = gw.String()
		log.Printf("host=gateway(%s)", host)
	}

	address := fmt.Sprintf("%s:%d", host, port)

	log.Printf("ping=%s podIP=%s nodeIP=%s gw=%v revision=%v", address, info.PodIP, info.NodeIP, gw, Revision)
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
		log.Printf("if=%s ips=%v\n", k, v)
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
			log.Printf("ping=%s status=failed err='%v'", address, err)
			success = false
			continue
		}

		if !success {
			log.Printf("ping=%s status=success\n", address)
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
