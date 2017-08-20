package main

import (
	"errors"
	"github.com/ww24/lirc-web-api/lirc"
	"log"
	"strconv"
	"strings"
	"time"
)

type Remote struct {
	name      string
	codes     []string
	lirc      *lirc.Client
	ch        chan *Command
	PrefixLen int
}

func discoverRemotes(lircClient *lirc.Client, ch chan *Command) (remotes map[string]*Remote, err error) {
	var list []string
	list, err = lircClient.List("")
	if err != nil {
		return
	}

	remotes = map[string]*Remote{}
	for _, v := range list {
		if v == "" {
			continue
		}
		remotes[v] = &Remote{
			name: v,
			lirc: lircClient,
			ch:   ch,
		}
		remotes[v].discoverCodes()

	}

	return
}

func codeToName(code string) string {
	if code == "" {
		return ""
	}
	if !strings.Contains(code, " ") {
		return ""
	}
	return strings.Split(code, " ")[1]
}

func (r *Remote) discoverCodes() error {
	list, err := r.lirc.List(r.name)
	if err != nil {
		return err
	}

	/**
	 * Remove duplicates
	 */
	names := map[string]struct{}{}

	for _, v := range list {
		if n := codeToName(v); n != "" {
			names[n] = struct{}{}
		}
	}

	r.codes = []string{}
	for k, _ := range names {
		r.codes = append(r.codes, k)
	}

	return nil
}

func (r *Remote) verifyCode(code string) string {
	for _, v := range r.codes {
		if strings.ToUpper(v) == strings.ToUpper(code) {
			return v
		}
	}
	return ""
}

func (r *Remote) SendOnce(code string) error {
	return r.Send(code, 0)
}

func (r *Remote) Send(code string, duration time.Duration) error {
	code = r.verifyCode(code)
	if code == "" {
		return errors.New("Unknown code " + code)
	}
	r.ch <- &Command{Remote: r.name, Code: code, Duration: duration}
	return nil
}

func (r *Remote) MQCallback(topic string, payload []byte) {
	dur, err := strconv.Atoi(string(payload))
	if err != nil {
		// 0 = SEND_ONCE
		dur = 0
	}

	if len(topic) <= r.PrefixLen {
		return
	}
	code := topic[r.PrefixLen:]
	log.Printf("[lirc] Sending %s to %s for %d ms (raw msg: %s[\"%s\"])", code, r.name, dur, topic, string(payload))

	err = r.Send(code, time.Millisecond*time.Duration(dur))
	if err != nil {
		log.Printf("[lirc] Unable to send code (%s/%s@%dms): %v", r.name, code, dur, err)
	}

}

func (r *Remote) Codes() []string {
	return r.codes
}
