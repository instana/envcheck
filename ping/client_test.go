package ping_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/instana/envcheck/ping"
)

func Test_success_on_match(t *testing.T) {
	t.Parallel()

	var response atomic.Value
	server, client, u := testServer(&response)
	defer server.Close()

	response.Store(u.Hostname())

	c := ping.New(client)

	err := c.Ping(u.Host, ping.DownwardInfo{NodeIP: u.Hostname()})
	if err != nil {
		t.Errorf("got <%v>, want nil", err)
	}
}

func Test_should_fail_on_mismatch(t *testing.T) {
	t.Parallel()
	var response atomic.Value
	response.Store("8.8.4.4")
	ts, client, u := testServer(&response)
	defer ts.Close()

	c := ping.New(client)

	err := c.Ping(u.Host, ping.DownwardInfo{NodeIP: u.Hostname()})
	if !strings.HasPrefix(err.Error(), "mismatch for nodeip received 8.8.4.4, wanted ") {
		t.Errorf("got <%v>, want mismatch error", err)
	}
}

func Test_should_fail_on_unbound_port(t *testing.T) {
	t.Parallel()
	var response atomic.Value
	ts, client, u := testServer(&response)
	defer ts.Close()

	c := ping.New(client)

	err := c.Ping("localhost:1035", ping.DownwardInfo{NodeIP: u.Hostname()})
	if !strings.HasSuffix(err.Error(), "connect: connection refused") {
		t.Errorf("got %v, want nil", err)
	}
}

func testServer(body *atomic.Value) (*httptest.Server, *http.Client, *url.URL) {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := body.Load().(string)
		_, err := fmt.Fprintf(w, h)
		if err != nil {
			http.Error(w, err.Error(), http.StatusTeapot)
		}
	}))
	ts.Start()

	client := ts.Client()
	client.Timeout = 1 * time.Second
	u, _ := url.Parse(ts.URL)
	return ts, client, u
}
