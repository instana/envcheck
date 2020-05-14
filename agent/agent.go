package agent

import "github.com/instana/envcheck/cluster"

// Size takes the summary details and calculates the appropriate resource limits
// for the Instana agent.
func Size(summary cluster.Summary) Limits {
	size := summary.Deployments + summary.Namespaces

	if size > 2000 {
		return Limits{
			CPULimit:      "2",
			CPURequest:    "500m",
			MemoryLimit:   "2Gi",
			MemoryRequest: "2Gi",
			Heap:          "800M",
		}
	} else if size > 1000 {
		return Limits{
			CPULimit:      "2",
			CPURequest:    "500m",
			MemoryLimit:   "1Gi",
			MemoryRequest: "1Gi",
			Heap:          "400M",
		}
	}

	return Limits{
		CPULimit:      "1.5",
		CPURequest:    "500m",
		MemoryLimit:   "512Mi",
		MemoryRequest: "512Mi",
		Heap:          "170M",
	}
}

// Limits describes the perscribed limits for the Instana agent.
type Limits struct {
	CPULimit      string
	CPURequest    string
	Heap          string
	MemoryLimit   string
	MemoryRequest string
}
