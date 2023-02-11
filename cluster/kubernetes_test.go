package cluster_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/instana/envcheck/cluster"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_InstanaLeader_should_return_leader(t *testing.T) {
	t.Parallel()
	endpoints := v1.EndpointsList{Items: []v1.Endpoints{instanaEndpoint(`{"holderIdentity":"instana-agent-hcdhs"}`)}}
	client := fake.NewSimpleClientset(&endpoints)
	query := cluster.NewQuery("localhost:1234", client.CoreV1(), client.AppsV1())
	leader, err := query.InstanaLeader()
	if err != nil {
		t.Errorf("query.InstanaLeader() err=%#v, want nil", err)
	}

	if leader != "instana-agent-hcdhs" {
		t.Errorf("query.InstanaLeader()=%s, want <instana-agent-hcdhs>", leader)
	}
}

func Test_InstanaLeader_should_return_invalid_format_if_json_invalid(t *testing.T) {
	t.Parallel()
	endpoints := v1.EndpointsList{Items: []v1.Endpoints{instanaEndpoint("foobar")}}
	client := fake.NewSimpleClientset(&endpoints)
	query := cluster.NewQuery("localhost:1234", client.CoreV1(), client.AppsV1())
	_, err := query.InstanaLeader()
	if err != cluster.ErrInvalidLeaseFormat {
		t.Errorf("query.InstanaLeader() err=%#v, want ErrInvalidLeaseFormat", err)
	}
}

func Test_InstanaLeader_should_return_leader_unknown_if_none_defined(t *testing.T) {
	t.Parallel()
	endpoints := v1.EndpointsList{Items: []v1.Endpoints{instanaEndpoint("")}}
	client := fake.NewSimpleClientset(&endpoints)
	query := cluster.NewQuery("localhost:1234", client.CoreV1(), client.AppsV1())
	_, err := query.InstanaLeader()
	if err != cluster.ErrLeaderUndefined {
		t.Errorf("query.InstanaLeader() err=%#v, want ErrLeaderUndefined", err)
	}
}

func Test_InstanaLeader_should_error_when_no_endpoint(t *testing.T) {
	t.Parallel()
	endpoints := v1.EndpointsList{Items: []v1.Endpoints{}}
	client := fake.NewSimpleClientset(&endpoints)
	query := cluster.NewQuery("localhost:1234", client.CoreV1(), client.AppsV1())
	_, err := query.InstanaLeader()
	_, ok := err.(*errors.StatusError)
	if !ok {
		t.Errorf("query.InstanaLeader() err=%#v, want StatusError NotFound", err)
	}
}

func Test_AllNodes(t *testing.T) {
	t.Parallel()
	items := []v1.Node{awsHost()}
	client := fake.NewSimpleClientset(&v1.NodeList{Items: items})
	query := cluster.NewQuery("localhost:1234", client.CoreV1(), client.AppsV1())

	all, err := query.AllNodes()
	if err != nil {
		t.Errorf("err=%v, want nil", err)
	}

	if len(all) != 1 {
		t.Errorf("len(all)=%v, want 1", len(all))
	}

	expected := cluster.NodeInfo{
		Name:         "ip-10-255-223-76.us-west-2.compute.internal",
		InstanceType: "r5.2xlarge",
		Zone:         "us-west-2b",
	}

	if !cmp.Equal(&expected, &all[0]) {
		t.Errorf("AllNodes()[0] mismatch (-want +got):\n%s", cmp.Diff(&expected, &all[0]))
	}
}

func Test_AllPods(t *testing.T) {
	t.Parallel()
	items := []v1.Pod{instanaAgent()}

	client := fake.NewSimpleClientset(&v1.PodList{Items: items})

	query := cluster.NewQuery("localhost:1234", client.CoreV1(), client.AppsV1())

	all, err := query.AllPods()
	if err != nil {
		t.Errorf("err=%v, want nil", err)
	}

	if len(all) != 1 {
		t.Errorf("len(all)=%v, want 1", len(all))
	}

	if query.Host() != "localhost:1234" {
		t.Errorf("host=%v, want localhost:1234", query.Host())
	}

	expected := cluster.PodInfo{
		IsRunning:  true,
		Name:       "agent-abc123",
		Namespace:  "instana-agent",
		Host:       "192.168.0.1",
		Owners:     map[string]string{"uid": "DaemonSet"},
		Containers: []cluster.ContainerInfo{{Name: "instana-agent", Image: "instana-agent:latest"}},
		Status:     "Running",
	}
	if !cmp.Equal(&expected, &all[0]) {
		t.Errorf("AllPods()[0] mismatch (-want +got):\n%s", cmp.Diff(&expected, &all[0]))
	}
}

func instanaEndpoint(value string) v1.Endpoints {
	annotations := map[string]string{
		"k8s.instana.io/clusterid": "3babd325-b451-40df-97a5-0398d0080fe8",
	}
	if value != "" {
		annotations["control-plane.alpha.kubernetes.io/leader"] = value
	}

	return v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "instana",
			Namespace:       "default",
			ResourceVersion: "1234",
			Annotations:     annotations,
		},
	}
}

func instanaAgent() v1.Pod {
	var container v1.Container
	container.Image = "instana-agent:latest"
	container.Name = "instana-agent"
	var instanaAgent v1.Pod
	instanaAgent.Name = "agent-abc123"
	instanaAgent.Namespace = "instana-agent"
	instanaAgent.OwnerReferences = append(instanaAgent.OwnerReferences, metav1.OwnerReference{Name: "uid", Kind: "DaemonSet"})
	instanaAgent.Status.HostIP = "192.168.0.1"
	instanaAgent.Status.Phase = v1.PodRunning
	instanaAgent.Spec.Containers = append(instanaAgent.Spec.Containers, container)
	return instanaAgent
}

func awsHost() v1.Node {
	var node v1.Node
	node.Name = "ip-10-255-223-76.us-west-2.compute.internal"
	node.Labels = map[string]string{
		"alpha.eksctl.io/cluster-name":             "k8s-infra-us-west-2",
		"alpha.eksctl.io/nodegroup-name":           "k8s-infra-us-west-2-private-beeinstant-aggregator-2",
		"beta.kubernetes.io/arch":                  "amd64",
		"beta.kubernetes.io/instance-type":         "r5.2xlarge",
		"beta.kubernetes.io/os":                    "linux",
		"eks.amazonaws.com/capacityType":           "ON_DEMAND",
		"eks.amazonaws.com/nodegroup":              "k8s-infra-us-west-2-private-beeinstant-aggregator-2",
		"eks.amazonaws.com/nodegroup-image":        "ami-083ae3afdae9db445",
		"failure-domain.beta.kubernetes.io/region": "us-west-2",
		"failure-domain.beta.kubernetes.io/zone":   "us-west-2b",
		"instanaGroup":                             "beeinstant-aggregator",
		"k8s.io/cloud-provider-aws":                "ee35cab3121e7a6e8e1422f6229646a3",
		"kubernetes.io/arch":                       "amd64",
		"kubernetes.io/hostname":                   "ip-10-255-223-76.instana.io",
		"kubernetes.io/os":                         "linux",
		"node.kubernetes.io/instance-type":         "r5.2xlarge",
		"topology.kubernetes.io/region":            "us-west-2",
		"topology.kubernetes.io/zone":              "us-west-2b",
	}
	node.Annotations = map[string]string{
		"node.alpha.kubernetes.io/ttl":                           "0",
		"volumes.kubernetes.io/controller-managed-attach-detach": "true",
	}
	return node
}
