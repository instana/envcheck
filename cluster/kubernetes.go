package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	appv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	typev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	// imports all auth methods for kubernetes go client.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// ErrLeaderUndefined is returned when the instana endpoint exists but no leader annotation exists.
	ErrLeaderUndefined = fmt.Errorf("endpoint found but leader undefined")
	// ErrInvalidLeaseFormat is returned when the leader annotation does not contain a valid LeaderLease.
	ErrInvalidLeaseFormat = fmt.Errorf("invalid lease format")
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

	return NewQuery(config.Host, clientset.CoreV1(), clientset.AppsV1()), nil
}

// Query is a query interface for the cluster.
type Query interface {
	// AllPods returns the list of pods from the related cluster.
	AllPods() ([]PodInfo, error)
	AllNodes() ([]NodeInfo, error)
	Host() string
	Time() time.Time
	InstanaLeader() (string, error)
}

// NewQuery allocates and returns a new Query.
func NewQuery(h string, cs typev1.CoreV1Interface, apps appv1.AppsV1Interface) *KubernetesQuery {
	return &KubernetesQuery{h, cs, apps}
}

// KubernetesQuery is a concrete Kubernetes client to query various cluster info.
type KubernetesQuery struct {
	host string
	core typev1.CoreV1Interface
	apps appv1.AppsV1Interface
}

// Time returns the current time.
func (q *KubernetesQuery) Time() time.Time {
	return time.Now()
}

// Host provides the host info for the cluster.
func (q *KubernetesQuery) Host() string {
	return q.host
}

// InstanaLeader returns the instana agent leader pod name.
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

// LeaderLease is the lease struct for the leader elector sidecar.
type LeaderLease struct {
	HolderIdentity string `json:"holderIdentity"`
	// {"holderIdentity":"instana-agent-hcdhs","leaseDurationSeconds":10,"acquireTime":"2020-06-03T19:54:57Z","renewTime":"2020-06-03T20:04:12Z","leaderTransitions":0}`
}

type NodeInfo struct {
	ContainerRuntime string
	InstanceType     string
	KernelVersion    string
	KubeletVersion   string
	Name             string
	OSImage          string
	ProxyVersion     string
	Zone             string
}

const limit = 250
const pauseTime = 50

func (q *KubernetesQuery) AllNodes() ([]NodeInfo, error) {
	var cont string
	var nodeList []NodeInfo

	for true {
		nodes, err := q.core.Nodes().List(context.TODO(), metav1.ListOptions{Limit: limit, Continue: cont})
		if err != nil {
			return nil, err
		}

		for _, node := range nodes.Items {
			nodeInfo := node.Status.NodeInfo
			labels := node.Labels

			info := NodeInfo{
				Name:             node.Name,
				ContainerRuntime: nodeInfo.ContainerRuntimeVersion,
				InstanceType:     labels["node.kubernetes.io/instance-type"],
				KernelVersion:    nodeInfo.KernelVersion,
				KubeletVersion:   nodeInfo.KubeletVersion,
				OSImage:          nodeInfo.OSImage,
				ProxyVersion:     nodeInfo.KubeProxyVersion,
				Zone:             labels["topology.kubernetes.io/zone"],
			}
			nodeList = append(nodeList, info)

		}

		cont = nodes.Continue
		if cont == "" {
			break
		}
		time.Sleep(pauseTime * time.Millisecond)
	}

	return nodeList, nil
}

type LinkedConfigMap struct {
	Name      string
	Namespace string
}

// AllPods retrieves all pod info from the cluster.
func (q *KubernetesQuery) AllPods() ([]PodInfo, error) {
	var cont string
	var podList []PodInfo
	namespaces := make(map[string]bool)

	for true {
		pods, err := q.core.Pods("").List(context.TODO(), metav1.ListOptions{Limit: limit, Continue: cont})
		if err != nil {
			return nil, err
		}

		for _, pod := range pods.Items {
			info := PodInfo{
				ChartVersion: pod.Labels["app.kubernetes.io/version"],
				Host:         pod.Status.HostIP,
				IsRunning:    pod.Status.Phase == v1.PodRunning,
				Name:         pod.Name,
				Namespace:    pod.Namespace,
				Owners:       make(map[string]string),
				Status:       string(pod.Status.Phase),
			}
			for _, owner := range pod.OwnerReferences {
				info.Owners[owner.Name] = owner.Kind
			}
			namespaces[pod.Namespace] = true

			var linkedConfigMaps []LinkedConfigMap
			for _, vol := range pod.Spec.Volumes {
				if vol.ConfigMap != nil {
					linkedConfigMaps = append(linkedConfigMaps, LinkedConfigMap{
						Name:      vol.ConfigMap.Name,
						Namespace: pod.Namespace,
					})
				}
			}

			var containers []ContainerInfo
			for _, container := range pod.Spec.Containers {
				containers = append(containers, ContainerInfo{
					Image: container.Image,
					Name:  container.Name,
				})
			}
			for _, status := range pod.Status.ContainerStatuses {
				info.Restarts += int(status.RestartCount)
			}
			info.Containers = containers
			info.LinkedConfigMaps = linkedConfigMaps
			podList = append(podList, info)
		}

		cont = pods.Continue
		if cont == "" {
			break
		}
		time.Sleep(pauseTime * time.Millisecond)
	}

	return podList, nil
}

// AgentEvent represents a single K8S event associated with the agent.
type AgentEvent struct {
	EventTime time.Time
	Reason    string
	Message   string
}

// AgentInfo provides general information relating to the agent.
type AgentInfo struct {
	Available    int32
	Desired      int32
	EventList    []AgentEvent
	Misscheduled int32
	Ready        int32
	Unavailable  int32
}

// AgentInfo queries the api-server for details about the Instana agent.
func (q *KubernetesQuery) AgentInfo(namespace string, name string) (*AgentInfo, error) {
	ds, err := q.apps.DaemonSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	uid := string(ds.UID)
	eventInterface := q.core.Events(namespace)
	selector := eventInterface.GetFieldSelector(&name, &namespace, nil, &uid)
	opts := metav1.ListOptions{
		FieldSelector: selector.String(),
	}
	list, err := eventInterface.List(context.TODO(), opts)
	if err != nil {
		return nil, err
	}

	var events []AgentEvent
	for _, v := range list.Items {
		var event = AgentEvent{
			EventTime: v.EventTime.Time,
			Reason:    v.Reason,
			Message:   v.Message,
		}
		events = append(events, event)
	}

	info := &AgentInfo{
		Available:    ds.Status.NumberAvailable,
		Desired:      ds.Status.DesiredNumberScheduled,
		EventList:    events,
		Misscheduled: ds.Status.NumberMisscheduled,
		Ready:        ds.Status.NumberReady,
		Unavailable:  ds.Status.NumberUnavailable,
	}
	return info, nil
}
