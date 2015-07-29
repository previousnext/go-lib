package ssh

import (
	"os"
)

func Exec(cmd string, user string, host string) error {
  // Ensure we have a port for the host.
  host, err := cleanHost(host)
  if err != nil {
		return err
	}

	config, err := newSshClientConfig(host, user)
	if err != nil {
		return err
	}

	session, err := config.NewSession(config.host)
	if err != nil {
		return err
	}

	// Output the command to standard out.
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	defer func() {
		session.Close()
	}()

	return session.Run(cmd)
}
