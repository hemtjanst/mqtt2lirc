package main

import (
	"context"
	"flag"
	"github.com/ww24/lirc-web-api/lirc"
	"lib.hemtjan.st/transport/mqtt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
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
	mqCfg := mqtt.MustFlags(flag.String, flag.Bool)
	flag.Parse()
	pfx := *flagTopicPrefix
	cfg := mqCfg()
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

	ctx, cancel := context.WithCancel(context.Background())
	tr, err := mqtt.New(ctx, cfg)
	if err != nil {
		log.Fatalf("Unable to create MQTT transport: %v\r\n", err)
	}

	wg := sync.WaitGroup{}

	for name, remote := range remotes {
		wg.Add(1)
		go func(name string, remote *Remote) {
			defer wg.Done()
			rPfx := pfx + name
			remote.PrefixLen = len(rPfx) + 1
			tr.Publish(rPfx, []byte(strings.Join(remote.Codes(), ",")), true)
			rmtCh := tr.SubscribeRaw(rPfx + "/#")
			for {
				cmd, open := <-rmtCh
				if !open {
					return
				}
				remote.MQCallback(cmd.TopicName, cmd.Payload)
			}

		}(name, remote)
	}
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		cancel()
	}()

	lircQueue(ch, lircClient, ctx)
}

func lircQueue(ch chan *Command, lirc *lirc.Client, ctx context.Context) {
	var err error
	for {
		select {
		case <-ctx.Done():
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
