package main

import (
	"io/ioutil"
	"log"
	"testing"

	"github.com/google/go-cmp/cmp"
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

func Test_parse_unknown_flag(t *testing.T) {
	t.Parallel()
	_, err := Parse([]string{"envcheckctl", "version", "-foobar"}, "", ioutil.Discard)
	if err.Error() != "flag provided but not defined: -foobar" {
		t.Errorf("err=%#v, want <flag provided but not defined: -foobar>", err)
	}
}

func Test_parse_flags(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		args   []string
		config *EnvcheckConfig
	}{
		"daemon":             {[]string{"envcheckctl", "daemon"}, &EnvcheckConfig{Subcommand: ApplyDaemon, AgentNamespace: "instana-agent"}},
		"inspect":            {[]string{"envcheckctl", "inspect"}, &EnvcheckConfig{Subcommand: InspectCluster}},
		"inspect offline":    {[]string{"envcheckctl", "inspect", "-podfile=foobar.json"}, &EnvcheckConfig{Subcommand: InspectCluster, Podfile: "foobar.json"}},
		"ping":               {[]string{"envcheckctl", "ping"}, &EnvcheckConfig{Subcommand: ApplyPinger, PingerNamespace: "default"}},
		"ping using gateway": {[]string{"envcheckctl", "ping", "-use-gateway"}, &EnvcheckConfig{Subcommand: ApplyPinger, PingerNamespace: "default", UseGateway: true}},
		"leader":             {[]string{"envcheckctl", "leader"}, &EnvcheckConfig{Subcommand: Leader}},
		"leader profile":     {[]string{"envcheckctl", "leader", "-profile"}, &EnvcheckConfig{Subcommand: Leader, Profile: true}},
		"version":            {[]string{"envcheckctl", "version"}, &EnvcheckConfig{Subcommand: PrintVersion}},
	}
	for name, tc := range cases {
		tc := tc
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
