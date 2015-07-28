package ssh

import (
	"errors"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
  "log"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// Helper function to execute an ssh session.
func runSSH(cmd string, user string, host string, forwarding bool) error {
  // Ensure we have a port for the host.
  host, err := cleanHost(host)
  if err != nil {
		return err
	}

	config, err := newSshClientConfig(host, user, "id_rsa_cirrus", forwarding)
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
		log.Printf("Session complete from %s", host)
	}()

	return session.Run("/bin/bash -c '"+cmd+"'")
}

// sshSession stores the open session and connection to execute a command.
type sshSession struct {
	// conn is the ssh client that started the session.
	conn *ssh.Client

	*ssh.Session
}

// Close closses the open ssh session and connection.
func (s *sshSession) Close() {
	s.Session.Close()
	s.conn.Close()
}

// sshClientConfig stores the configuration
// and the ssh agent to forward authentication requests
type sshClientConfig struct {
	// agent is the connection to the ssh agent
	agent agent.Agent

	// host to connect to
	host string

	*ssh.ClientConfig
}

// newSshClientConfig initializes the ssh configuration.
// It connects with the ssh agent when agent forwarding is enabled.
func newSshClientConfig(host string, userName, identity string, agentForwarding bool) (*sshClientConfig, error) {
	var (
		config *sshClientConfig
		err    error
	)

	if agentForwarding {
		config, err = newSshAgentConfig(userName)
	} else {
		config, err = newSshDefaultConfig(userName, identity)
	}

	if config != nil {
		config.host = host
	}
	return config, err
}

// newSshAgentConfig initializes the configuration to talk with an ssh agent.
func newSshAgentConfig(userName string) (*sshClientConfig, error) {
	agent, err := newAgent()
	if err != nil {
		return nil, err
	}

	config, err := sshAgentConfig(userName, agent)
	if err != nil {
		return nil, err
	}

	return &sshClientConfig{
		agent:        agent,
		ClientConfig: config,
	}, nil
}

// newSshDefaultConfig initializes the configuration to use an ideitity file.
func newSshDefaultConfig(userName, identity string) (*sshClientConfig, error) {
	config, err := sshDefaultConfig(userName, identity)
	if err != nil {
		return nil, err
	}

	return &sshClientConfig{ClientConfig: config}, nil
}

// NewSession creates a new ssh session with the host.
// It forwards authentication to the agent when it's configured.
func (s *sshClientConfig) NewSession(host string) (*sshSession, error) {
	conn, err := ssh.Dial("tcp", host, s.ClientConfig)
	if err != nil {
		return nil, err
	}

	if s.agent != nil {
		if err := agent.ForwardToAgent(conn, s.agent); err != nil {
			return nil, err
		}
	}

	session, err := conn.NewSession()
	if s.agent != nil {
		err = agent.RequestAgentForwarding(session)
	}

	return &sshSession{
		conn:    conn,
		Session: session,
	}, err
}

// newAgent connects with the SSH agent in the to forward authentication requests.
func newAgent() (agent.Agent, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, errors.New("Unable to connect to the ssh agent. Please, check that SSH_AUTH_SOCK is set and the ssh agent is running")
	}

	conn, err := net.Dial("unix", sock)
	if err != nil {
		return nil, err
	}

	return agent.NewClient(conn), nil
}

// sshAgentConfig creates a new configuration for the ssh client
// with the signatures from the ssh agent.
func sshAgentConfig(userName string, a agent.Agent) (*ssh.ClientConfig, error) {
	signers, err := a.Signers()
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signers...),
		},
	}, nil
}

// sshDefaultConfig returns the SSH client config for the connection
func sshDefaultConfig(userName, identity string) (*ssh.ClientConfig, error) {
	contents, err := loadIdentity(userName, identity)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(contents)
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User: userName,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}, nil
}

// loadIdentity returns the private key file's contents
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
