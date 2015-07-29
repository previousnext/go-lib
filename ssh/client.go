package ssh

import (
	"errors"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

type sshClientConfig struct {
	agent agent.Agent
	host string
	*ssh.ClientConfig
}

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

func newSshClientConfig(host string, userName string) (*sshClientConfig, error) {
	config, err := newSshAgentConfig(userName)
	if config != nil {
		config.host = host
	}
	return config, err
}

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
