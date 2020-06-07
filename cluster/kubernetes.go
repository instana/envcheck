package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	typev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	// imports all auth methods for kubernetes go client.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	ErrLeaderUndefined    = fmt.Errorf("Endpoint found but leader undefined")
	ErrInvalidLeaseFormat = fmt.Errorf("Invalid lease format")
)

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

	return NewQuery(config.Host, clientset.CoreV1()), nil
}

// Query is a query interface for the cluster.
type Query interface {
	// AllPods returns the list of pods from the related cluster.
	AllPods() ([]PodInfo, error)
	Host() string
	Time() time.Time
	InstanaLeader() (string, error)
}

// NewQuery allocates and returns a new Query.
func NewQuery(h string, cs typev1.CoreV1Interface) *KubernetesQuery {
	return &KubernetesQuery{h, cs}
}

// KubernetesQuery is a concrete Kubernetes client to query various cluster info.
type KubernetesQuery struct {
	host string
	core typev1.CoreV1Interface
}

// Time returns the current time.
func (q *KubernetesQuery) Time() time.Time {
	return time.Now()
}

// Host provides the host info for the cluster.
func (q *KubernetesQuery) Host() string {
	return q.host
}

func (q *KubernetesQuery) InstanaLeader() (string, error) {
	ep, err := q.core.Endpoints("default").Get(context.TODO(), "instana", metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	v, ok := ep.Annotations["control-plane.alpha.kubernetes.io/leader"]
	if !ok {
		return "", ErrLeaderUndefined
	}

	var lease LeaderLease
	err = json.Unmarshal([]byte(v), &lease)
	if err != nil {
		return "", ErrInvalidLeaseFormat
	}

	return lease.HolderIdentity, nil
}

// {"holderIdentity":"instana-agent-hcdhs","leaseDurationSeconds":10,"acquireTime":"2020-06-03T19:54:57Z","renewTime":"2020-06-03T20:04:12Z","leaderTransitions":0}`
type LeaderLease struct {
	HolderIdentity string `json:"holderIdentity"`
}

// AllPods retrieves all pod info from the cluster.
func (q *KubernetesQuery) AllPods() ([]PodInfo, error) {
	var cont string
	var podList []PodInfo
	namespaces := make(map[string]bool)

	for true {
		pods, err := q.core.Pods("").List(context.TODO(), metav1.ListOptions{Limit: 100, Continue: cont})
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
