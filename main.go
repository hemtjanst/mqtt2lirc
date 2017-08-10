package main

import (
	"flag"
	mqtt "github.com/bonan/mqtt_flags"
	"github.com/ww24/lirc-web-api/lirc"
	"log"
	"os"
	"strings"
)

var (
	flagSocket      = flag.String("lirc.socket", "/var/run/lirc/lircd", "Path to lircd unix socket")
	flagTopicPrefix = flag.String("mqtt.prefix", "remote/", "Prefix for MQTT topics")
)

func main() {
	flag.Parse()
	pfx := *flagTopicPrefix

	lircClient, err := lirc.New(*flagSocket)
	if err != nil {
		log.Fatalf("Unable to open connection to lirc: %v\r\n", err)
	}
	defer lircClient.Close()

	mq, err := mqtt.NewPersistantMqtt(mqtt.ClientConfig{
		Logger: log.New(os.Stdout, "[MQTT] ", 0),
	})
	if err != nil {
		log.Fatalf("Unable to initialize MQTT: %v\r\n", err)
	}
	defer mq.Stop()

	remotes, err := discoverRemotes(lircClient)
	if err != nil {
		log.Fatalf("Unable to interface with LIRC: %v\r\n", err)
	}

	remoteNames := []string{}
	for name, remote := range remotes {
		rPfx := pfx + name
		remote.PrefixLen = len(rPfx) + 1
		mq.Publish(
			rPfx, // Topic
			1,    // QoS
			true, // Retain
			[]byte(strings.Join(remote.Codes(), ",")),
		)
		mq.Subscribe(rPfx+"/#", 1, remote.MQCallback)
		remoteNames = append(remoteNames, name)
	}

	mq.Publish(
		strings.TrimRight(pfx, "/"), // Topic
		1,    // QoS
		true, // Retain
		[]byte(strings.Join(remoteNames, ",")),
	)

	mq.Start()
}
