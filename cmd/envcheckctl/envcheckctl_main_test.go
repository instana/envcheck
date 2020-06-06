package main

import (
	"io/ioutil"
	"reflect"
	"testing"
)

func Test_parse_flags(t *testing.T) {
	cases := map[string]struct {
		args   []string
		config *EnvcheckConfig
		err    error
	}{
		"no subcommand":      {[]string{"envcheckctl"}, nil, ErrNoSubcommand},
		"invalid subcommand": {[]string{"envcheckctl", "foobar"}, nil, ErrNoSubcommand},
		"daemon":             {[]string{"envcheckctl", "daemon"}, &EnvcheckConfig{Subcommand: ApplyDaemon, AgentNamespace: "instana-agent"}, nil},
		"inspect":            {[]string{"envcheckctl", "inspect"}, &EnvcheckConfig{Subcommand: InspectCluster, AgentNamespace: "instana-agent"}, nil},
		"inspect offline":    {[]string{"envcheckctl", "inspect", "-podfile=foobar.json"}, &EnvcheckConfig{Subcommand: InspectCluster, AgentNamespace: "instana-agent", Podfile: "foobar.json"}, nil},
		"ping":               {[]string{"envcheckctl", "ping"}, &EnvcheckConfig{Subcommand: ApplyPinger, PingerNamespace: "default"}, nil},
		"ping using gateway": {[]string{"envcheckctl", "ping", "-use-gateway"}, &EnvcheckConfig{Subcommand: ApplyPinger, PingerNamespace: "default", UseGateway: true}, nil},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			actual, err := Parse(tc.args, "", ioutil.Discard)
			if err != err {
				t.Errorf("err=%v, want %v", err, tc.err)
			}

			if !reflect.DeepEqual(actual, tc.config) {
				t.Errorf("config=%#v, want %#v", actual, tc.config)
			}
		})
	}
}
