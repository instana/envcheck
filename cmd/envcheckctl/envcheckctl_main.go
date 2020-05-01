package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

func Exec(kubeconfig string) {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalln(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err.Error())
	}

	cluster := &ClusterInfo{
		Name: config.Host,
	}

	log.Printf("cluster=%v\n", cluster.Name)
	log.Println("Collecting pod details. This vary depending on the cluster.")
	pods, err := AllPods(clientset)
	if err != nil {
		log.Fatalf("error retrieving pods: %v\n", err)
	}
	cluster.Pods = pods
	cluster.PodCount = len(pods)
	log.Printf("Total of %d pods\n", len(cluster.Pods))

	w, err := os.Create("cluster-info.json")
	if err != nil {
		log.Fatalln(err)
	}
	defer w.Close()

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	err = enc.Encode(cluster)
	if err != nil {
		log.Fatalln(err)
	}
}

func AllPods(clientset *kubernetes.Clientset) ([]PodInfo, error) {
	var cont string
	var podList []PodInfo
	namespaces := make(map[string]bool)

	for true {
		pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{Limit: 100, Continue: cont})
		if err != nil {
			return nil, err
		}

		for _, pod := range pods.Items {
			info := PodInfo{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				Host:      pod.Status.HostIP,
				Owners:    make(map[string]string),
			}
			for _, owner := range pod.OwnerReferences {
				info.Owners[owner.Name] = owner.Kind
			}
			namespaces[pod.Namespace] = true

			var containers []ContainerInfo
			for _, container := range pod.Spec.Containers {
				containers = append(containers, ContainerInfo{
					Image: container.Image,
					Name:  container.Name,
				})
			}
			info.Containers = containers
			podList = append(podList, info)
		}

		cont = pods.Continue
		if cont == "" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	return podList, nil
}

func main() {
	var kubeconfig string

	if home := homeDir(); home != "" {
		flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	Exec(kubeconfig)
}

func homeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}

type ClusterInfo struct {
	Name     string
	PodCount int
	Pods     []PodInfo
	Version  string
}

type PodInfo struct {
	Containers []ContainerInfo
	Host       string
	Name       string
	Namespace  string
	Owners     map[string]string
}

type ContainerInfo struct {
	Name  string
	Image string
}
