package main

import (
	"flag"
	mq "github.com/eclipse/paho.mqtt.golang"
	"github.com/hemtjanst/hemtjanst/messaging/flagmqtt"
	"github.com/ww24/lirc-web-api/lirc"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	flagSocket      = flag.String("lirc.socket", "/var/run/lirc/lircd", "Path to lircd unix socket")
	flagTopicPrefix = flag.String("mqtt.prefix", "remote/", "Prefix for MQTT topics")
)

type Command struct {
	Remote   string
	Code     string
	Duration time.Duration
}

func main() {
	flag.Parse()
	pfx := *flagTopicPrefix

	lircClient, err := lirc.New(*flagSocket)
	if err != nil {
		log.Fatalf("Unable to open connection to lirc: %v\r\n", err)
	}
	defer lircClient.Close()

	ch := make(chan *Command, 32)

	remotes, err := discoverRemotes(lircClient, ch)
	if err != nil {
		log.Fatalf("Unable to interface with LIRC: %v\r\n", err)
	}

	mqttClient, err := flagmqtt.NewPersistentMqtt(flagmqtt.ClientConfig{
		OnConnectHandler: func(client mq.Client) {
			remoteNames := []string{}
			for name, remote := range remotes {
				rPfx := pfx + name
				remote.PrefixLen = len(rPfx) + 1
				client.Publish(
					rPfx, // Topic
					1,    // QoS
					true, // Retain
					[]byte(strings.Join(remote.Codes(), ",")),
				)
				rmt := remote
				tok := client.Subscribe(rPfx+"/#", 1, func(client mq.Client, msg mq.Message) {
					log.Printf("MQTT[%s]: %s", msg.Topic(), string(msg.Payload()))
					rmt.MQCallback(msg.Topic(), msg.Payload())
				})
				tok.Wait()
				if tok.Error() != nil {
					log.Printf("Error subscribing to %s/#: %v", rPfx, tok.Error())
				}
				remoteNames = append(remoteNames, name)
			}

			client.Publish(
				strings.TrimRight(pfx, "/"), // Topic
				1,    // QoS
				true, // Retain
				[]byte(strings.Join(remoteNames, ",")),
			)
		},
	})
	if err != nil {
		log.Fatalf("Unable to initialize MQTT: %v\r\n", err)
	}
	defer mqttClient.Disconnect(250)

	connTok := mqttClient.Connect()
	connTok.Wait()
	if connTok.Error() != nil {
		log.Fatalf("Unable to connect to MQTT: %v", connTok.Error())
	}

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	lircQueue(ch, lircClient, quit)
}

func lircQueue(ch chan *Command, lirc *lirc.Client, quit chan os.Signal) {
	var err error
	for {
		select {
		case <-quit:
			return
		case cmd := <-ch:
			if cmd == nil {
				return
			}

			if cmd.Duration == 0 {
				err = lirc.SendOnce(cmd.Remote, cmd.Code)
				if err != nil {
					log.Printf("Error sending %s to %s: %v", cmd.Remote, cmd.Code, err)
				}
				continue
			}

			err = lirc.SendStart(cmd.Remote, cmd.Code)
			if err != nil {
				log.Printf("Error sending %s to %s (for %d ms): %v", cmd.Remote, cmd.Code, cmd.Duration/time.Millisecond, err)
			}

			<-time.After(cmd.Duration)
			err = lirc.SendStop(cmd.Remote, cmd.Code)
			if err != nil {
				log.Printf("Calling SendStop on %s/%s after %d ms: %v", cmd.Remote, cmd.Code, cmd.Duration/time.Millisecond, err)
			}
		}
	}
}
