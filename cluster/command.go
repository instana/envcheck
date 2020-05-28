package cluster

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/tools/clientcmd"
)

// NewCommand builds a new KubernetesCommand implementation with the given kubeconfig.
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

type Command interface {
	CreateDaemon(DaemonConfig) error
	CreatePingerr(PingerConfig) error
	CreateService(ServiceConfig) error
}

type KubernetesCommand struct {
	appsv1.AppsV1Interface
}

func (kc *KubernetesCommand) CreateDaemon(config DaemonConfig) error {
	_, err := kc.DaemonSets(config.Namespace).Create(context.TODO(), Daemon(config), metav1.CreateOptions{})
	return err
}
func (kc *KubernetesCommand) CreatePinger(config PingerConfig) error {
	_, err := kc.DaemonSets(config.Namespace).Create(context.TODO(), Pinger(config), metav1.CreateOptions{})
	return err
}

func (kc *KubernetesCommand) CreateService(ServiceConfig) error {
	return nil
}
