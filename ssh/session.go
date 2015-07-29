package ssh

import (
  "golang.org/x/crypto/ssh"
)

type sshSession struct {
	conn *ssh.Client
	*ssh.Session
}

func (s *sshSession) Close() {
	s.Session.Close()
	s.conn.Close()
}
