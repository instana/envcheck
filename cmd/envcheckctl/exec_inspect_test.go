package main

import (
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/instana/envcheck/cluster"
)

func Test_LoadInfo_reads_empty_ClusterInfo(t *testing.T) {
	t.Parallel()
	r := strings.NewReader(`{"Name": "https://gke.gcloud.com:8443","PodCount": 0,"Pods": []}`)
	info, err := LoadInfo(r)
	if err != nil {
		t.Errorf("err=%v, want nil", err)
	}

	expected := &cluster.Info{Name: "https://gke.gcloud.com:8443", Pods: []cluster.PodInfo{}}
	if !cmp.Equal(expected, info) {
		t.Errorf("LoadInfo() mismatch (-want +got)\n%s", cmp.Diff(expected, info))
	}
}

func Test_LoadInfo_errors_with_invalid_json(t *testing.T) {
	t.Parallel()
	r := strings.NewReader(`{"bloop"`)
	_, err := LoadInfo(r)
	if err == nil {
		t.Error("err=nil, want json err")
	}
}

func Test_QueryLive_should_count_pods_correctly(t *testing.T) {
	t.Parallel()
	query := &stubQuery{}
	info, _ := QueryLive(query)

	if info.PodCount != 2 {
		t.Errorf("info.PodCount=%v, want 2", info.PodCount)
	}
}

func Test_QueryLive_should_associate_pods_correctly(t *testing.T) {
	t.Parallel()
	query := &stubQuery{}
	info, _ := QueryLive(query)

	if len(info.Pods) != 2 {
		t.Errorf("len(info.Pods)=%v, want 2", info.PodCount)
	}
}

type stubQuery struct {
	ts time.Time
}

func (q *stubQuery) ServerVersion() (string, error) {
	return "v1.23.14-eks-ffeb93d", nil
}

func (q *stubQuery) InstanaLeader() (string, error) {
	return "instana-agent-hcdhs", nil
}

func (q *stubQuery) Time() time.Time {
	return q.ts
}

func (q *stubQuery) Host() string {
	return "https://localhost:8443"
}

func (q *stubQuery) AllNodes() ([]cluster.NodeInfo, error) {
	return []cluster.NodeInfo{}, nil
}

func (q *stubQuery) AllPods() ([]cluster.PodInfo, error) {
	pods := []cluster.PodInfo{
		{
			Host: "192.168.253.101",
			Containers: []cluster.ContainerInfo{
				{
					Name:  "instana-agent",
					Image: "instana-agent/instana-agent:latest",
				},
			},
			IsRunning: true,
			Name:      "instana-agent-xyz123",
			Namespace: "instana-agent",
			Owners: map[string]string{
				"instana-agent": "DaemonSet",
			},
		},
		{
			Host: "192.168.253.102",
			Containers: []cluster.ContainerInfo{
				{
					Name:  "instana-agent",
					Image: "instana-agent/instana-agent:latest",
				},
			},
			IsRunning: true,
			Name:      "instana-agent-123xyz",
			Namespace: "instana-agent",
			Owners: map[string]string{
				"instana-agent": "DaemonSet",
			},
		},
	}
	return pods, nil
}
