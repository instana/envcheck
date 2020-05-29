package main

import (
	"expvar"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
)

var (
	// Revision is the Git commit SHA injected at compile time.
	Revision = "dev"
)

// Exec is the primary execution for the daemon application.
func Exec(address string, info DownwardInfo) error {
	log.Printf("daemon=%s pod=%s/%s listen=%s podIP=%s nodeIP=%s", Revision, info.Namespace, info.Name, address, info.PodIP, info.NodeIP)
	publish("address", address)
	publish("name", info.Name)
	publish("namespace", info.Namespace)
	publish("nodeIP", info.NodeIP)
	publish("podIP", info.PodIP)

	ifs, err := MapInterfaces()
	if err != nil {
		return err
	}

	for k, v := range ifs {
		log.Printf("pod=%s/%s if=%s ips=%v\n", info.Namespace, info.Name, k, v)
	}

	http.HandleFunc("/ping", PingHandler(info))

	return http.ListenAndServe(address, nil)
}

// MapInterfaces provides a map of interfaces to IP addresses.
func MapInterfaces() (map[string][]string, error) {
	m := make(map[string][]string)
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, adapter := range interfaces {
		addresses, err := adapter.Addrs()
		if err != nil {
			return nil, err
		}

		var ips []string
		for _, address := range addresses {
			ips = append(ips, address.String())
		}
		m[adapter.Name] = ips
	}

	return m, nil
}

// PingHandler handles pings from the pinger client.
func PingHandler(info DownwardInfo) func(w http.ResponseWriter, r *http.Request) {
	b := []byte(info.NodeIP)

	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}
}

func main() {
	var address string
	var downward DownwardInfo

	flag.StringVar(&downward.Name, "name", os.Getenv("NAME"), "name of this pod.")
	flag.StringVar(&downward.Namespace, "namespace", os.Getenv("NAMESPACE"), "namespace this service is running in.")
	flag.StringVar(&downward.NodeIP, "nodeip", os.Getenv("NODEIP"), "node IP this service is running on.")
	flag.StringVar(&downward.PodIP, "podip", os.Getenv("PODIP"), "pod IP this service is running on.")
	flag.StringVar(&address, "address", os.Getenv("ADDRESS"), "listening address for this service to bind on.")
	flag.Parse()

	err := Exec(address, downward)

	if err != nil {
		log.Printf("status=shutdown pod=%s/%s error='%v'\n", downward.Namespace, downward.Name, err)
		os.Exit(-1)
	}
}

func publish(key string, value string) {
	expvar.NewString(key).Set(value)
}

// DownwardInfo is the data injected into the pod from the downward API.
type DownwardInfo struct {
	Name      string
	Namespace string
	NodeIP    string
	PodIP     string
}
