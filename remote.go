package main

import "github.com/ww24/lirc-web-api/lirc"

type Remote struct {
	name  string
	codes []string
	lirc  lirc.ClientAPI
}

func discoverRemotes(lircClient lirc.ClientAPI) (remotes map[string]*Remote, err error) {
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
		}
		remotes[v].discoverCodes()

	}

	return
}

func (r *Remote) discoverCodes() error {
	list, err := r.lirc.List(r.name)
	if err != nil {
		return err
	}
	names := map[string]struct{}{}

	for _, v := range list {
		names[v] = nil
	}

	r.codes = make([]string, len(names))
	for k, _ := range names {
		r.codes = append(r.codes, k)
	}

	return nil
}
