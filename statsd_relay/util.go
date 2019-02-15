package statsd_relay

import (
	"log"
)

func LogError(err error) {
	if err != nil {
		log.Print(err)
	}
}

func ExitOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
