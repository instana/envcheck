package envcheck

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func New(client *http.Client) *Client {
	return &Client{
		client: client,
	}
}

type Client struct {
	client *http.Client
}

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

type DownwardInfo struct {
	Name      string
	Namespace string
	NodeIP    string
	PodIP     string
}
