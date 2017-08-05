package main

import (
	"flag"
	"github.com/ww24/lirc-web-api/lirc"
	"github.com/yosssi/gmq/mqtt/client"
	"log"
	mqtt2 "github.com/bonan/mqtt2lirc/mqtt"
)

var (
	flagSocket       = flag.String("lirc.socket", "/var/run/lirc/lircd", "Path to lircd unix socket")
	flagTopicPrefix  = flag.String("mqtt.prefix", "remote/", "Prefix for MQTT topics")
)

func main() {
	flag.Parse()

	lircClient, err := lirc.New(*flagSocket)
	if err != nil {
		log.Fatalf("Unable to open connection to lirc: %v\r\n", err)
	}

	mqtt, err := mqtt2.NewPersistantMqtt(mqtt2.ClientConfig{})
	if err != nil {
		log.Fatalf("Unable to initialize MQTT: %v\r\n", err)
	}

	remotes, err := discoverRemotes(lircClient)
	if err != nil {
		log.Fatalf("Unable to interface with LIRC: %v\r\n", err)
	}

}
