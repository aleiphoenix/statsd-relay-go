package statsd_relay

import (
	"bytes"
	"fmt"
	"net"
)

type UDPInput struct {
	Host       string
	Port       int
	Connection *net.UDPConn
}

func (u *UDPInput) Init() {
	url := fmt.Sprintf("%s:%d", u.Host, u.Port)
	addr, err := net.ResolveUDPAddr("udp", url)
	LogError(err)
	u.Connection, err = net.ListenUDP("udp", addr)
	LogError(err)
}

func (u *UDPInput) Drain(queue chan []byte) {
	for {
		buf := make([]byte, 65535)
		_, _, err := u.Connection.ReadFromUDP(buf)
		LogError(err)
		queue <- bytes.Trim(buf, "\x00")
	}
}
