package cluster

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	// imports all auth methods for kubernetes go client.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

// Query is a query interface for the cluster.
type Query interface {
	// AllPods returns the list of pods from the related cluster.
	AllPods() ([]PodInfo, error)
	Host() string
}

// New builds a new KubernetesQuery implementation with the given kubeconfig.
func New(kubeconfig string) (*KubernetesQuery, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &KubernetesQuery{config.Host, clientset}, nil
}

// KubernetesQuery is a concrete Kubernetes client to query various cluster info.
type KubernetesQuery struct {
	host string
	*kubernetes.Clientset
}

// Host provides the host info for the cluster.
func (q *KubernetesQuery) Host() string {
	return q.host
}

// AllPods retrieves all pod info from the cluster.
func (q *KubernetesQuery) AllPods() ([]PodInfo, error) {
	var cont string
	var podList []PodInfo
	namespaces := make(map[string]bool)

	for true {
		pods, err := q.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{Limit: 100, Continue: cont})
		if err != nil {
			return nil, err
		}

		for _, pod := range pods.Items {
			info := PodInfo{
				IsRunning: pod.Status.Phase == v1.PodRunning,
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
