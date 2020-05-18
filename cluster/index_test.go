package cluster_test

import (
	"reflect"
	"testing"

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
	ns1 := cluster.PodInfo{Namespace: "one", Owners: owner(cluster.ReplicaSet)}
	ns2 := cluster.PodInfo{Namespace: "two", Owners: owner(cluster.ReplicaSet)}

	index := cluster.NewIndex()
	index.Each(ns1)
	index.Each(ns2)
	actual := index.Summary()
	expected := cluster.Summary{
		Deployments: 1, Namespaces: 2, Nodes: 1, Pods: 2,
	}
	if !reflect.DeepEqual(&actual, &expected) {
		t.Errorf("summary=%#v, want %#v", actual, expected)
	}
}

func Test_Each_container_increments_containers(t *testing.T) {
	container1 := cluster.PodInfo{Host: "one", Name: "pod-1", Containers: []cluster.ContainerInfo{container("instana")}, Owners: owner(cluster.ReplicaSet)}
	container2 := cluster.PodInfo{Host: "two", Name: "pod-2", Containers: []cluster.ContainerInfo{container("instana"), container("leader")}, Owners: owner(cluster.ReplicaSet)}

	index := cluster.NewIndex()
	index.Each(container1)
	index.Each(container2)
	actual := index.Summary()
	expected := cluster.Summary{
		Containers: 3, Deployments: 1, Namespaces: 1, Nodes: 2, Pods: 2,
	}
	if !reflect.DeepEqual(&actual, &expected) {
		t.Errorf("summary=%#v, want %#v", actual, expected)
	}
}

func container(name string) cluster.ContainerInfo {
	return cluster.ContainerInfo{
		Name:  name,
		Image: "instana:latest",
	}
}

func Test_Each_increments_unique_hosts(t *testing.T) {
	host1 := cluster.PodInfo{Host: "one", Name: "pod-1", Owners: owner(cluster.ReplicaSet)}
	host2 := cluster.PodInfo{Host: "two", Name: "pod-2", Owners: owner(cluster.ReplicaSet)}

	index := cluster.NewIndex()
	index.Each(host1)
	index.Each(host2)
	actual := index.Summary()
	expected := cluster.Summary{
		Deployments: 1, Namespaces: 1, Nodes: 2, Pods: 2,
	}
	if !reflect.DeepEqual(&actual, &expected) {
		t.Errorf("summary=%#v, want %#v", actual, expected)
	}
}

func Test_Each_increments_by_owner_type(t *testing.T) {
	testCases := map[string]struct {
		pod     cluster.PodInfo
		summary cluster.Summary
	}{
		"should increment daemonset":   {cluster.PodInfo{Owners: owner(cluster.DaemonSet)}, cluster.Summary{DaemonSets: 1, Namespaces: 1, Nodes: 1, Pods: 1}},
		"should increment deployment":  {cluster.PodInfo{Owners: owner(cluster.ReplicaSet)}, cluster.Summary{Deployments: 1, Namespaces: 1, Nodes: 1, Pods: 1}},
		"should increment statefulset": {cluster.PodInfo{Owners: owner(cluster.StatefulSet)}, cluster.Summary{StatefulSets: 1, Namespaces: 1, Nodes: 1, Pods: 1}},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			index := cluster.NewIndex()
			index.Each(tc.pod)
			actual := index.Summary()
			if !reflect.DeepEqual(&actual, &tc.summary) {
				t.Errorf("summary=%#v, want %#v", actual, tc.summary)
			}
		})
	}
}
