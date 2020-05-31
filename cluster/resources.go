package cluster

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DaemonSetName = "envchecker"
	ManagedBy     = "envcheckctl"
	PingerName    = "pinger"
)

type DaemonConfig struct {
	Namespace string
	Image     string
	Version   string
	Host      string
	Port      int32
}

func (dc *DaemonConfig) Address() string {
	return fmt.Sprintf("%s:%d", dc.Host, dc.Port)
}

func Daemon(config DaemonConfig) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DaemonSetName,
			Namespace: config.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/managed-by": ManagedBy,
					"app.kubernetes.io/name":       DaemonSetName,
					"app.kubernetes.io/version":    config.Version,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": ManagedBy,
						"app.kubernetes.io/name":       DaemonSetName,
						"app.kubernetes.io/version":    config.Version,
					},
				},
				Spec: v1.PodSpec{
					HostNetwork: true,
					Containers: []v1.Container{
						{
							Name:            DaemonSetName,
							Image:           config.Image,
							ImagePullPolicy: v1.PullAlways,
							Resources: ResourceRequirements(Resources{
								RequestCPU:    "5m",
								RequestMemory: "20Mi",
								LimitCPU:      "100m",
								LimitMemory:   "512Mi",
							}),
							Env: []v1.EnvVar{
								FieldPath("NAME", "metadata.name"),
								FieldPath("NAMESPACE", "metadata.namespace"),
								FieldPath("NODEIP", "status.hostIP"),
								FieldPath("PODIP", "status.podIP"),
								{Name: "ADDRESS", Value: config.Address()},
							},
							Ports: []v1.ContainerPort{
								{
									Name:          "http",
									Protocol:      v1.ProtocolTCP,
									ContainerPort: config.Port,
								},
							},
						},
					},
				},
			},
		},
	}
}

type PingerConfig struct {
	Namespace string
	Image     string
	Version   string
	Host      string
	Port      int32
}

func Pinger(config PingerConfig) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PingerName,
			Namespace: config.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/managed-by": ManagedBy,
					"app.kubernetes.io/name":       PingerName,
					"app.kubernetes.io/version":    config.Version,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app.kubernetes.io/managed-by": ManagedBy,
						"app.kubernetes.io/name":       PingerName,
						"app.kubernetes.io/version":    config.Version,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:            PingerName,
							Image:           config.Image,
							ImagePullPolicy: v1.PullAlways,
							Resources: ResourceRequirements(Resources{
								RequestCPU:    "5m",
								RequestMemory: "20Mi",
								LimitCPU:      "100m",
								LimitMemory:   "512Mi",
							}),
							Env: []v1.EnvVar{
								FieldPath("NAME", "metadata.name"),
								FieldPath("NAMESPACE", "metadata.namespace"),
								FieldPath("NODEIP", "status.hostIP"),
								FieldPath("PODIP", "status.podIP"),
								PingHost(config.Host),
								{Name: "PINGPORT", Value: fmt.Sprintf("%d", config.Port)},
							},
						},
					},
				},
			},
		},
	}
}

type ServiceConfig struct {
	Namespace string
}

func PingHost(host string) v1.EnvVar {
	const name = "PINGHOST"
	if host == "" {
		return FieldPath(name, "status.hostIP")
	}

	return v1.EnvVar{Name: name, Value: host}
}

func FieldPath(name, path string) v1.EnvVar {
	return v1.EnvVar{
		Name: name,
		ValueFrom: &v1.EnvVarSource{
			FieldRef: &v1.ObjectFieldSelector{
				FieldPath: path,
			},
		},
	}
}

type Resources struct {
	RequestCPU    string
	RequestMemory string
	LimitCPU      string
	LimitMemory   string
}

func ResourceRequirements(resources Resources) v1.ResourceRequirements {
	return v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse(resources.RequestCPU),
			v1.ResourceMemory: resource.MustParse(resources.RequestMemory),
		},
		Limits: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse(resources.LimitCPU),
			v1.ResourceMemory: resource.MustParse(resources.LimitMemory),
		},
	}
}
