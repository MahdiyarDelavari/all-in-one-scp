package transfer

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)

// BrowserShell manages one live SSH shell session for the web UI.
type BrowserShell struct {
	mu      sync.Mutex
	client  *ssh.Client
	session *ssh.Session
	stdin   io.WriteCloser
}

func NewBrowserShell() *BrowserShell {
	return &BrowserShell{}
}

func (s *BrowserShell) Start(config ConnectionConfig, emit func(string)) error {
	_ = s.Close()

	client, err := connectSSH(config)
	if err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		client.Close()
		return fmt.Errorf("create SSH session: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		return fmt.Errorf("open SSH stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		client.Close()
		return fmt.Errorf("open SSH stderr pipe: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		return fmt.Errorf("open SSH stdin pipe: %w", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm", 24, 120, modes); err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return fmt.Errorf("request SSH PTY: %w", err)
	}

	if err := session.Shell(); err != nil {
		stdin.Close()
		session.Close()
		client.Close()
		return fmt.Errorf("start SSH shell: %w", err)
	}

	s.mu.Lock()
	s.client = client
	s.session = session
	s.stdin = stdin
	s.mu.Unlock()

	go s.pipeOutput(stdout, emit)
	go s.pipeOutput(stderr, emit)
	go s.waitForExit(session, client, stdin, emit)

	return nil
}

func (s *BrowserShell) SendLine(line string) error {
	s.mu.Lock()
	stdin := s.stdin
	s.mu.Unlock()

	if stdin == nil {
		return fmt.Errorf("SSH session is not connected")
	}

	if _, err := io.WriteString(stdin, line+"\n"); err != nil {
		return fmt.Errorf("send SSH command: %w", err)
	}

	return nil
}

func (s *BrowserShell) Close() error {
	s.mu.Lock()
	client := s.client
	session := s.session
	stdin := s.stdin
	s.client = nil
	s.session = nil
	s.stdin = nil
	s.mu.Unlock()

	if stdin != nil {
		_ = stdin.Close()
	}

	if session != nil {
		_ = session.Close()
	}

	if client != nil {
		return client.Close()
	}

	return nil
}

func (s *BrowserShell) pipeOutput(reader io.Reader, emit func(string)) {
	buffer := make([]byte, 4096)

	for {
		count, err := reader.Read(buffer)
		if count > 0 {
			chunk := sanitizeTerminalChunk(string(buffer[:count]))
			if chunk != "" {
				emit(chunk)
			}
		}

		if err != nil {
			return
		}
	}
}

func (s *BrowserShell) waitForExit(session *ssh.Session, client *ssh.Client, stdin io.WriteCloser, emit func(string)) {
	err := session.Wait()

	s.mu.Lock()
	if s.session == session {
		s.client = nil
		s.session = nil
		s.stdin = nil
	}
	s.mu.Unlock()

	_ = stdin.Close()
	_ = session.Close()
	_ = client.Close()

	if err != nil && !strings.Contains(err.Error(), "EOF") {
		emit("\n[SSH session closed: " + err.Error() + "]\n")
		return
	}

	emit("\n[SSH session closed]\n")
}

func sanitizeTerminalChunk(chunk string) string {
	chunk = strings.ReplaceAll(chunk, "\r\n", "\n")
	chunk = strings.ReplaceAll(chunk, "\r", "")
	chunk = ansiPattern.ReplaceAllString(chunk, "")
	return chunk
}
