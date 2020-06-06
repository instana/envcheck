package cluster

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DaemonSetName is the resource name for the envchecker daemon set.
	DaemonSetName = "envchecker"
	// ManagedBy is the resource managed by identifier.
	ManagedBy = "envcheckctl"
	// PingerName is the resource name for the pinger daemon set.
	PingerName = "pinger"
)

// DaemonConfig is the conffiguration for the envchecker daemon set.
type DaemonConfig struct {
	Namespace string
	Image     string
	Version   string
	Host      string
	Port      int32
}

// Address provides the combined host and port pair as an address.
func (dc *DaemonConfig) Address() string {
	return fmt.Sprintf("%s:%d", dc.Host, dc.Port)
}

const (
	LabelManagedBy = "app.kubernetes.io/managed-by"
	LabelName      = "app.kubernetes.io/name"
	LabelVersion   = "app.kubernetes.io/version"
)

// Daemon creates the envchecker daemon set resource from the provided DaemonConfig.
func Daemon(config DaemonConfig) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DaemonSetName,
			Namespace: config.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					LabelName:    DaemonSetName,
					LabelVersion: config.Version,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						LabelManagedBy: ManagedBy,
						LabelName:      DaemonSetName,
						LabelVersion:   config.Version,
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

// PingerConfig is the input configuration for the pinger DaemonSet.
type PingerConfig struct {
	Namespace  string
	Image      string
	Version    string
	Host       string
	Port       int32
	UseGateway bool
}

// Pinger creates the pinger DaemonSet resource from the provided PingerConfig.
func Pinger(config PingerConfig) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PingerName,
			Namespace: config.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					LabelName:    PingerName,
					LabelVersion: config.Version,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						LabelManagedBy: ManagedBy,
						LabelName:      PingerName,
						LabelVersion:   config.Version,
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
								PingHost(config.Host, config.UseGateway),
								{Name: "PINGPORT", Value: fmt.Sprintf("%d", config.Port)},
							},
						},
					},
				},
			},
		},
	}
}

// PingHost outputs the "PINGHOST" env var key value based on host and useGateway.
func PingHost(host string, useGateway bool) v1.EnvVar {
	const name = "PINGHOST"
	if useGateway {
		return v1.EnvVar{Name: name, Value: ""}
	}

	if host == "" {
		return FieldPath(name, "status.hostIP")
	}

	return v1.EnvVar{Name: name, Value: host}
}

// FieldPath returns a single env var based on the given field name and path.
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

// Resources is the system resource contraints associated with an entity.
type Resources struct {
	RequestCPU    string
	RequestMemory string
	LimitCPU      string
	LimitMemory   string
}

// ResourceRequirements builds a kubernetes resource requirements from the provided Resources.
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
