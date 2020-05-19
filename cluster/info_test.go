package cluster_test

import (
	"testing"

	"github.com/instana/envcheck/cluster"
)

func Test_Apply(t *testing.T) {
	testCases := map[string]struct {
		info  cluster.Info
		count int
	}{
		"empty":   {info: cluster.Info{}, count: 0},
		"one pod": {info: cluster.Info{Pods: []cluster.PodInfo{{}}}, count: 1},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := &counter{}
			tc.info.Apply(c)
			if c.count != tc.count {
				t.Errorf("count=%v, want %v", c.count, tc.count)
			}
		})
	}
}

type counter struct {
	count int
}

func (c *counter) Each(_ cluster.PodInfo) {
	c.count++
}
