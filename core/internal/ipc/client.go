package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
)

type Client struct {
	conn net.Conn
	scan *bufio.Scanner
}

func Dial(socketPath string) (*Client, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", socketPath, err)
	}

	scan := bufio.NewScanner(conn)
	scan.Buffer(make([]byte, 64*1024), 1024*1024)
	if !scan.Scan() {
		conn.Close()
		return nil, fmt.Errorf("read capabilities: %w", scan.Err())
	}

	return &Client{conn: conn, scan: scan}, nil
}

func (c *Client) Call(req Request) (*Response[any], error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if _, err := c.conn.Write(append(data, '\n')); err != nil {
		return nil, err
	}
	if !c.scan.Scan() {
		return nil, fmt.Errorf("read response: %w", c.scan.Err())
	}
	var resp Response[any]
	if err := json.Unmarshal(c.scan.Bytes(), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Close() error { return c.conn.Close() }
