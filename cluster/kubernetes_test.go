package cluster_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/instana/envcheck/cluster"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_AllPods(t *testing.T) {
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

	items := []v1.Pod{instanaAgent}

	client := fake.NewSimpleClientset(&v1.PodList{Items: items})

	query := cluster.NewQuery("localhost:1234", client.CoreV1())

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
	}
	if !cmp.Equal(&expected, &all[0]) {
		t.Errorf("AllPods()[0] mismatch (-want +got):\n%s", cmp.Diff(&expected, &all[0]))
	}
}
