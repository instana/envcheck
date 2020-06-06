package main

import (
	"io/ioutil"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/instana/envcheck/cluster"
)

func init() {
	log.SetOutput(ioutil.Discard)
}

func Test_parse_no_subcommand(t *testing.T) {
	t.Parallel()
	_, err := Parse([]string{"envcheckctl"}, "", ioutil.Discard)
	if err != ErrNoSubcommand {
		t.Errorf("err=%v, want ErrNoSubcommand", err)
	}
}

func Test_parse_unknown_subcommand(t *testing.T) {
	t.Parallel()
	_, err := Parse([]string{"envcheckctl", "foobar"}, "", ioutil.Discard)
	if err != ErrUnknownSubcommand {
		t.Errorf("err=%v, want ErrUnknownSubcommand", err)
	}
}

func Test_parse_flags(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		args   []string
		config *EnvcheckConfig
	}{
		"daemon":             {[]string{"envcheckctl", "daemon"}, &EnvcheckConfig{Subcommand: ApplyDaemon, AgentNamespace: "instana-agent"}},
		"inspect":            {[]string{"envcheckctl", "inspect"}, &EnvcheckConfig{Subcommand: InspectCluster, AgentNamespace: "instana-agent"}},
		"inspect offline":    {[]string{"envcheckctl", "inspect", "-podfile=foobar.json"}, &EnvcheckConfig{Subcommand: InspectCluster, AgentNamespace: "instana-agent", Podfile: "foobar.json"}},
		"ping":               {[]string{"envcheckctl", "ping"}, &EnvcheckConfig{Subcommand: ApplyPinger, PingerNamespace: "default"}},
		"ping using gateway": {[]string{"envcheckctl", "ping", "-use-gateway"}, &EnvcheckConfig{Subcommand: ApplyPinger, PingerNamespace: "default", UseGateway: true}},
		"version":            {[]string{"envcheckctl", "version"}, &EnvcheckConfig{Subcommand: PrintVersion}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual, err := Parse(tc.args, "", ioutil.Discard)
			if err != nil {
				t.Errorf("err=%v, want nil", err)
			}

			if !cmp.Equal(tc.config, actual) {
				t.Errorf("Parse() mismatch (-want +got)\n%s", cmp.Diff(tc.config, actual))
			}
		})
	}
}

func Test_config_with_podfile_is_offline(t *testing.T) {
	t.Parallel()
	config := EnvcheckConfig{Podfile: "bloop.json"}
	if config.IsLive() {
		t.Errorf("config.IsLive()=%v, want false", config.IsLive())
	}
}

func Test_config_without_podfile_is_online(t *testing.T) {
	t.Parallel()
	config := EnvcheckConfig{Podfile: ""}
	if !config.IsLive() {
		t.Errorf("config.IsLive()=%v, want true", config.IsLive())
	}
}

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

func (q *stubQuery) Time() time.Time {
	return q.ts
}

func (q *stubQuery) Host() string {
	return "https://localhost:8443"
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
