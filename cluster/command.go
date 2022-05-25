package cluster

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
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

	return &KubernetesCommand{clientset.AppsV1(), clientset.CoreV1()}, nil
}

// Command provides an interface for creating envcheck entities in a cluster.
type Command interface {
	CreateDaemon(DaemonConfig) error
	CreatePinger(PingerConfig) error
	CreateService(DaemonConfig) error
}

// KubernetesCommand is a k8s implementation of the Command interface.
type KubernetesCommand struct {
	appsv1.AppsV1Interface
	corev1.CoreV1Interface
}

// CreateDaemon creates an envchecker daemonset in the current K8S environment.
func (kc *KubernetesCommand) CreateDaemon(config DaemonConfig) error {
	_, err := kc.DaemonSets(config.Namespace).Create(context.TODO(), Daemon(config), metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		_, err = kc.DaemonSets(config.Namespace).Update(context.TODO(), Daemon(config), metav1.UpdateOptions{})
	}
	return err
}

// CreateService creates an envchecker service in the current K8S environment.
func (kc *KubernetesCommand) CreateService(config DaemonConfig) error {
	svc := Service(config)
	_, err := kc.Services(config.Namespace).Create(context.TODO(), svc, metav1.CreateOptions{})
	return err
}

// CreatePinger creates a pinger daemonset in the current K8S environment.
func (kc *KubernetesCommand) CreatePinger(config PingerConfig) error {
	_, err := kc.DaemonSets(config.Namespace).Create(context.TODO(), Pinger(config), metav1.CreateOptions{})
	if err != nil && errors.IsAlreadyExists(err) {
		_, err = kc.DaemonSets(config.Namespace).Update(context.TODO(), Pinger(config), metav1.UpdateOptions{})
	}
	return err
}
