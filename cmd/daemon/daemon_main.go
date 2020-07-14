package main

import (
	"expvar"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/instana/envcheck/network"
)

var (
	// Revision is the Git commit SHA injected at compile time.
	Revision = "dev"
)

// Exec is the primary execution for the daemon application.
func Exec(address string, info DownwardInfo) error {
	log.SetFlags(log.LUTC | log.Lmsgprefix | log.LstdFlags)
	log.SetPrefix(fmt.Sprintf("pod=%s/%s ", info.Namespace, info.Name))

	log.Printf("listen=%s podIP=%s nodeIP=%s kubeApi=%s daemon=%s", address, info.PodIP, info.NodeIP, info.KubeAPI, Revision)
	publish("address", address)
	publish("name", info.Name)
	publish("namespace", info.Namespace)
	publish("nodeIP", info.NodeIP)
	publish("podIP", info.PodIP)
	publish("kubeAPI", info.KubeAPI)

	ifs, err := network.MapInterfaces()
	if err != nil {
		return err
	}

	for k, v := range ifs {
		log.Printf("if=%s ips=%v\n", k, v)
	}

	http.HandleFunc("/ping", PingHandler(info))

	return http.ListenAndServe(address, nil)
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
	var apiHost string
	var apiPort string
	var address string
	var downward DownwardInfo

	flag.StringVar(&downward.Name, "name", os.Getenv("NAME"), "name of this pod.")
	flag.StringVar(&downward.Namespace, "namespace", os.Getenv("NAMESPACE"), "namespace this service is running in.")
	flag.StringVar(&downward.NodeIP, "nodeip", os.Getenv("NODEIP"), "node IP this service is running on.")
	flag.StringVar(&downward.PodIP, "podip", os.Getenv("PODIP"), "pod IP this service is running on.")
	flag.StringVar(&address, "address", os.Getenv("ADDRESS"), "listening address for this service to bind on.")
	flag.StringVar(&apiHost, "kubehost", os.Getenv("KUBERNETES_SERVICE_HOST"), "kube api host")
	flag.StringVar(&apiPort, "kubeport", os.Getenv("KUBERNETES_SERVICE_PORT"), "kube api port")
	flag.Parse()

	downward.KubeAPI = fmt.Sprintf("%s:%s", apiHost, apiPort)

	err := Exec(address, downward)

	if err != nil {
		log.Printf("status=shutdown error='%v'\n", err)
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
	KubeAPI   string
}
