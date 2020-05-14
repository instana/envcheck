package ping

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// New creates a pinger client.
func New(client *http.Client) *Client {
	return &Client{
		client: client,
	}
}

// Client encapsulates a HTTP pinger client.
type Client struct {
	client *http.Client
}

// Ping requests the ping daemon pods ping end-point.
func (c *Client) Ping(address string, info DownwardInfo) error {
	resp, err := c.client.Get(fmt.Sprintf("http://%s/ping", address))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		return err
	}

	actual := buf.String()
	if actual == "" || info.NodeIP != actual {
		return fmt.Errorf("mismatch for nodeip received %v, wanted %v", actual, info.NodeIP)
	}
	return nil
}

// DownwardInfo is the data injected into the pod from the downward API.
type DownwardInfo struct {
	Name      string
	Namespace string
	NodeIP    string
	PodIP     string
}
