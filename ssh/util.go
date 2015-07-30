package ssh

import (
	"io/ioutil"
	"net"
	"os/user"
	"path/filepath"
	"strings"
	"time"
	"errors"
)

func loadIdentity(userName, identity string) ([]byte, error) {
	if filepath.Dir(identity) == "." {
		u, err := user.Current()
		if err != nil {
			return nil, err
		}
		identity = filepath.Join(u.HomeDir, ".ssh", identity)
	}

	return ioutil.ReadFile(identity)
}

func cleanHost(host string) (string, error) {
	h, port, err := net.SplitHostPort(host)
	if err != nil {
		if !strings.Contains(err.Error(), "missing port in address") {
			return "", err
		}
		port = "22"
		h = host
	}
	if port == "" {
		port = "22"
	}
	return net.JoinHostPort(h, port), nil
}

func wait(host string) error {
	times := 0
	for {
		_, err := net.Dial("tcp", host)
		if err != nil {
			// We need to make sure we have not created a loop.
			if times >= 10 {
				return errors.New("Cannot connect to the host after "+string(times)+" attemps")
			}
		} else {
			// The port has become available.
			return nil
		}
		time.Sleep(10000 * time.Millisecond)
	}
}
