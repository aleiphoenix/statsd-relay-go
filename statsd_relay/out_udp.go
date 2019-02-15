package statsd_relay

import (
	"fmt"
	// "log"
	"net"
)

type UdpOutput struct {
	Host       string
	Port       int
	LocalAddr  *net.UDPAddr
	RemoteAddr *net.UDPAddr
	Connection *net.UDPConn
}

func (o *UdpOutput) Init() {
	var err error
	url := fmt.Sprintf("%s:%d", o.Host, o.Port)
	o.RemoteAddr, err = net.ResolveUDPAddr("udp", url)
	ExitOnError(err)
	o.LocalAddr, err = net.ResolveUDPAddr("udp", "")
	ExitOnError(err)
	o.Connection, err = net.ListenUDP("udp", o.LocalAddr)
	ExitOnError(err)
}

func (o *UdpOutput) Write(q chan []byte) {
	for m := range q {
		// log.Printf("udp out: %s\n", string(m))
		_, err := o.Connection.WriteToUDP(m, o.RemoteAddr)
		LogError(err)
	}
}
