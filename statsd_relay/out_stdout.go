package statsd_relay

import (
	"log"
	"os"
	"strings"
)

type StdoutOutput struct {
	Logger *log.Logger
}

func (o *StdoutOutput) Init() {
	o.Logger = log.New(
		os.Stdout, "",
		log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
}

func (o *StdoutOutput) Write(q chan []byte) {
	for s := range q {
		o.Logger.Println(strings.TrimSpace(string(s)))
	}
}
