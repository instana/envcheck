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
		"daemon":             {[]string{"envcheckctl", "daemon"}, &EnvcheckConfig{ApplyDaemon: true, AgentNamespace: "instana-agent"}, nil},
		"inspect":            {[]string{"envcheckctl", "inspect"}, &EnvcheckConfig{AgentNamespace: "instana-agent", IsLive: true}, nil},
		"ping":               {[]string{"envcheckctl", "ping"}, &EnvcheckConfig{ApplyPinger: true, PingerNamespace: "default"}, nil},
		"ping using gateway": {[]string{"envcheckctl", "ping", "-use-gateway"}, &EnvcheckConfig{ApplyPinger: true, PingerNamespace: "default", UseGateway: true}, nil},
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
