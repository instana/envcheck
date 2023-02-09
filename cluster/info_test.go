package cluster_test

import (
	"testing"

	"github.com/instana/envcheck/cluster"
)

func Test_Apply(t *testing.T) {
	testCases := map[string]struct {
		info  cluster.Info
		pods  int
		nodes int
	}{
		"empty":    {info: cluster.Info{}, pods: 0},
		"one pod":  {info: cluster.Info{Pods: []cluster.PodInfo{{}}}, pods: 1},
		"one node": {info: cluster.Info{Nodes: []cluster.NodeInfo{{}}}, nodes: 1},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := &counter{}
			tc.info.Apply(c)
			if c.pods != tc.pods {
				t.Errorf("pods=%v, want %v", c.pods, tc.pods)
			}
			if c.nodes != tc.nodes {
				t.Errorf("nodes=%v, want %v", c.nodes, tc.nodes)
			}
		})
	}
}

type counter struct {
	pods  int
	nodes int
}

func (c *counter) EachPod(_ cluster.PodInfo) {
	c.pods++
}

func (c *counter) EachNode(_ cluster.NodeInfo) {
	c.nodes++
}
