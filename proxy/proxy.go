package proxy

import (
	"github.com/sandertv/go-raknet"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Proxy struct {
	log *logrus.Logger

	proxyAddress      string
	downstreamAddress string

	id atomic.Int32

	clientsMu sync.Mutex
	clients   map[int]*Client

	listener *raknet.Listener
	start    sync.Once
}

func New(log *logrus.Logger, proxyAddress, downstreamAddress string) *Proxy {
	return &Proxy{
		proxyAddress:      proxyAddress,
		downstreamAddress: downstreamAddress,
		log:               log,
		clients:           make(map[int]*Client),
	}
}

func (p *Proxy) Start() {
	p.start.Do(func() {
		p.log.Infof("listening on %v", p.proxyAddress)

		listener, err := raknet.Listen(p.proxyAddress)
		if err != nil {
			p.log.Fatalf("failed to listen on %v: %v", p.proxyAddress, err)
		}
		p.listener = listener

		go func() {
			for {
				time.Sleep(5 * time.Second)
				p.syncPongData()
			}
		}()

		p.accept()
	})
}

func (p *Proxy) syncPongData() {
	pong, err := raknet.Ping(p.downstreamAddress)
	if err != nil {
		p.log.Errorf("failed to ping upstream: %v", err)
		return
	}

	p.listener.PongData(pong)
}

func (p *Proxy) accept() {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			p.log.Errorf("failed to accept connection: %v", err)
			continue
		}
		go p.onClientConnect(conn)
	}
}

func (p *Proxy) onClientConnect(conn net.Conn) {
	p.log.Infof("client connected from %v", conn.RemoteAddr())

	remote, err := raknet.Dial(p.downstreamAddress)
	if err != nil {
		p.log.Errorf("failed to connect to upstream: %v", err)
		_ = conn.Close()
		return
	}

	id := int(p.id.Add(1))

	client := NewClient(p, id, p.log.WithFields(logrus.Fields{
		"client": id,
		"addr":   conn.RemoteAddr(),
	}), conn, remote)

	p.clientsMu.Lock()
	p.clients[id] = client
	p.clientsMu.Unlock()

	client.Start()
}

func (p *Proxy) Close() error {
	p.clientsMu.Lock()
	defer p.clientsMu.Unlock()
	for _, client := range p.clients {
		client.Close()
	}
	return p.listener.Close()
}

func (p *Proxy) removeClient(id int) {
	p.clientsMu.Lock()
	delete(p.clients, id)
	p.clientsMu.Unlock()
}
