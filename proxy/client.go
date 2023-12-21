package proxy

import (
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"sync/atomic"
)

type Client struct {
	proxy      *Proxy
	id         int
	log        logrus.FieldLogger
	conn       net.Conn
	remoteConn net.Conn
	running    atomic.Bool
	close      sync.Once

	upstreamBuf   []byte
	downstreamBuf []byte
}

func NewClient(proxy *Proxy, id int, log logrus.FieldLogger, conn net.Conn, remoteConn net.Conn) *Client {
	return &Client{
		id:         id,
		proxy:      proxy,
		log:        log,
		conn:       conn,
		remoteConn: remoteConn,
	}
}

func (c *Client) Start() {
	c.running.Store(true)

	c.upstreamBuf = make([]byte, 1024*1024*4)
	c.downstreamBuf = make([]byte, 1024*1024*4)

	go c.handleRemote()
	c.handleClient()
}

func (c *Client) handleClient() {
	defer c.Close()

	for c.running.Load() {
		n, err := c.conn.Read(c.upstreamBuf)
		if err != nil {
			c.log.Errorf("failed to read from client: %v", err)
			return
		}

		_, err = c.remoteConn.Write(c.upstreamBuf[:n])
		if err != nil {
			c.log.Errorf("failed to write to remote: %v", err)
			return
		}
	}
}

func (c *Client) handleRemote() {
	defer c.Close()

	for c.running.Load() {
		n, err := c.remoteConn.Read(c.downstreamBuf)
		if err != nil {
			c.log.Errorf("failed to read from remote: %v", err)
			return
		}

		_, err = c.conn.Write(c.downstreamBuf[:n])
		if err != nil {
			c.log.Errorf("failed to write to client: %v", err)
			return
		}
	}
}

func (c *Client) Close() {
	c.close.Do(func() {
		c.running.Store(false)
		_ = c.conn.Close()
		_ = c.remoteConn.Close()
		c.proxy.removeClient(c.id)
	})
}
