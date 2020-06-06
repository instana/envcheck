package agent_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/instana/envcheck/agent"
	"github.com/instana/envcheck/cluster"
)

func Test_should_size_environment_correctly(t *testing.T) {
	td := []struct {
		Name          string
		CPURequest    string
		CPULimit      string
		MemoryRequest string
		MemoryLimit   string
		Heap          string
		Summary       cluster.Summary
	}{
		{"small cluster", "500m", "1.5", "512Mi", "512Mi", "170M", smallCluster()},
		{"medium cluster", "500m", "2", "1Gi", "1Gi", "400M", mediumCluster()},
		{"large cluster", "500m", "2", "2Gi", "2Gi", "800M", largeCluster()},
	}

	for _, tc := range td {
		t.Run(tc.Name, func(t *testing.T) {
			actual := agent.Size(tc.Summary)
			expected := agent.Limits{
				CPULimit:      tc.CPULimit,
				CPURequest:    tc.CPURequest,
				MemoryLimit:   tc.MemoryLimit,
				MemoryRequest: tc.MemoryRequest,
				Heap:          tc.Heap,
			}
			if !cmp.Equal(expected, actual) {
				t.Errorf("Size(Summary) mismatch (-want +got) \n %s", cmp.Diff(expected, actual))
			}
		})
	}
}

func smallCluster() cluster.Summary {
	return cluster.Summary{
		Containers:   12,
		DaemonSets:   3,
		Deployments:  2,
		Namespaces:   3,
		Nodes:        3,
		Pods:         12,
		Running:      11,
		StatefulSets: 0,
	}
}

func mediumCluster() cluster.Summary {
	return cluster.Summary{
		Containers:   12,
		DaemonSets:   3,
		Deployments:  1000,
		Namespaces:   300,
		Nodes:        100,
		Pods:         3000,
		Running:      3000,
		StatefulSets: 0,
	}
}

func largeCluster() cluster.Summary {
	return cluster.Summary{
		Containers:   12,
		DaemonSets:   3,
		Deployments:  2000,
		Namespaces:   500,
		Nodes:        100,
		Pods:         6000,
		Running:      6000,
		StatefulSets: 0,
	}
}
