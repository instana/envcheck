package cluster

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// NewCommand allocates and returns a new Command.
func NewCommand(kubeconfig string) (*KubernetesCommand, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &KubernetesCommand{clientset.AppsV1()}, nil
}

// Command provides an interface for creating envcheck entities in a cluster.
type Command interface {
	CreateDaemon(DaemonConfig) error
	CreatePinger(PingerConfig) error
}

// KubernetesCommand is a k8s implementation of the Command interface.
type KubernetesCommand struct {
	appsv1.AppsV1Interface
}

// CreateDaemon applies a envchecker daemonset config to the current K8S environment.
func (kc *KubernetesCommand) CreateDaemon(config DaemonConfig) error {
	_, err := kc.DaemonSets(config.Namespace).Create(context.TODO(), Daemon(config), metav1.CreateOptions{})
	return err
}

// CreatePinger applies a pinger daemonset config to the current K8S environment.
func (kc *KubernetesCommand) CreatePinger(config PingerConfig) error {
	_, err := kc.DaemonSets(config.Namespace).Create(context.TODO(), Pinger(config), metav1.CreateOptions{})
	return err
}
