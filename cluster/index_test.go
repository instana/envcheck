package cluster_test

import (
	"github.com/gogunit/gunit"

	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/instana/envcheck/cluster"
)

type SummaryBuilder struct {
	cluster.Summary
}

func newSummaryBuilder(Containers int,
	DaemonSets int,
	Deployments int,
	Namespaces int,
	Nodes int,
	Pods int,
	Running int,
	StatefulSets int) SummaryBuilder {
	return SummaryBuilder{
		cluster.Summary{
			DaemonSets:   DaemonSets,
			Deployments:  Deployments,
			Namespaces:   Namespaces,
			Nodes:        Nodes,
			Pods:         Pods,
			Running:      Running,
			StatefulSets: StatefulSets,
		},
	}
}

func owner(t string) map[string]string {
	return map[string]string{
		"uid": t,
	}
}

func Test_Each_increments_unique_namespaces(t *testing.T) {
	t.Parallel()
	ns1 := cluster.PodInfo{Host: "nod01", Namespace: "one", Owners: owner(cluster.ReplicaSet)}
	ns2 := cluster.PodInfo{Host: "nod01", Namespace: "two", Owners: owner(cluster.ReplicaSet)}

	index := cluster.NewIndex()
	index.EachPod(ns1)
	index.EachPod(ns2)

	actual := index.Summary()
	expected := cluster.Summary{
		Deployments: 1, Namespaces: 2, Nodes: 1, Pods: 2,
	}
	if !cmp.Equal(&actual, &expected) {
		t.Errorf("Summary() mismatch (-want +got)\n%s", cmp.Diff(expected, actual))
	}
}

func Test_Each_container_increments_containers(t *testing.T) {
	t.Parallel()
	// unnamed container
	container1 := cluster.PodInfo{Host: "one", Name: "pod-1", Containers: []cluster.ContainerInfo{container("")}, Owners: owner(cluster.ReplicaSet)}
	// named container with sidecar
	container2 := cluster.PodInfo{Host: "two", Name: "pod-2", Containers: []cluster.ContainerInfo{container("instana"), container("leader")}, Owners: owner(cluster.ReplicaSet)}

	index := cluster.NewIndex()
	index.EachPod(container1)
	index.EachPod(container2)
	actual := index.Summary()
	expected := cluster.Summary{
		Containers: 3, Deployments: 1, Namespaces: 1, Nodes: 2, Pods: 2,
	}
	if !cmp.Equal(&actual, &expected) {
		t.Errorf("Summary() mismatch (-want +got)\n%s", cmp.Diff(expected, actual))
	}
}

func container(name string) cluster.ContainerInfo {
	return cluster.ContainerInfo{
		Name:  name,
		Image: "instana:latest",
	}
}

func Test_Each_increments_unique_hosts(t *testing.T) {
	t.Parallel()
	host1 := cluster.PodInfo{Host: "one", Name: "pod-1", Owners: owner(cluster.ReplicaSet)}
	host2 := cluster.PodInfo{Host: "two", Name: "pod-2", Owners: owner(cluster.ReplicaSet)}
	unscheduled := cluster.PodInfo{Host: "", Name: "pod-2", Owners: owner(cluster.ReplicaSet)}

	index := cluster.NewIndex()
	index.EachPod(host1)
	index.EachPod(host2)
	index.EachPod(unscheduled)

	actual := index.Summary()
	expected := cluster.Summary{
		Deployments: 1, Namespaces: 1, Nodes: 2, Pods: 2,
	}
	if !cmp.Equal(&actual, &expected) {
		t.Errorf("Summary() mismatch (-want +got)\n%s", cmp.Diff(expected, actual))
	}
}

func Test_Each_increments_by_owner_type(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		pod     cluster.PodInfo
		summary cluster.Summary
	}{
		"should increment daemonset":   {cluster.PodInfo{Host: "node01", Owners: owner(cluster.DaemonSet)}, cluster.Summary{DaemonSets: 1, Namespaces: 1, Nodes: 1, Pods: 1}},
		"should increment deployment":  {cluster.PodInfo{Host: "node01", Owners: owner(cluster.ReplicaSet)}, cluster.Summary{Deployments: 1, Namespaces: 1, Nodes: 1, Pods: 1}},
		"should increment statefulset": {cluster.PodInfo{Host: "node01", Owners: owner(cluster.StatefulSet)}, cluster.Summary{StatefulSets: 1, Namespaces: 1, Nodes: 1, Pods: 1}},
		"should increment running":     {cluster.PodInfo{Host: "node01", Owners: owner(cluster.ReplicaSet), IsRunning: true}, cluster.Summary{Deployments: 1, Namespaces: 1, Nodes: 1, Pods: 1, Running: 1}},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			index := cluster.NewIndex()
			index.EachPod(tc.pod)
			actual := index.Summary()
			if !cmp.Equal(&actual, &tc.summary) {
				t.Errorf("Summary() mismatch (-want +got)\n%s", cmp.Diff(tc.summary, actual))
			}
		})
	}
}

func Test_Count_Pod_Status(t *testing.T) {
	var hosts []cluster.PodInfo
	hosts = append(hosts, cluster.PodInfo{Host: "one", Name: "pod-1", Owners: owner(cluster.DaemonSet)})
	hosts = append(hosts, cluster.PodInfo{Host: "two", Name: "pod-2", Owners: owner(cluster.DaemonSet)})
	hosts = append(hosts, cluster.PodInfo{Host: "three", Name: "pod-3", Owners: owner(cluster.ReplicaSet)})
	hosts = append(hosts, cluster.PodInfo{Host: "four", Name: "pod-4", Owners: owner(cluster.DaemonSet)})

	index := cluster.NewIndex()
	for _, host := range hosts {
		index.EachPod(host)
	}
	gunit.Number(t, index.PodStatus[""]).EqualTo(4)
}

func Test_IsCNIPlugin(t *testing.T) {
	td := map[string]struct {
		expected bool
	}{
		"instana-agent":         {false},
		"kube-flannel-ds-kdwch": {true},
		"aws-node":              {true},
	}

	for n, tc := range td {
		t.Run(n, func(t *testing.T) {
			gunit.Struct(t, cluster.IsCNIPlugin(n)).EqualTo(tc.expected)
		})
	}
}
