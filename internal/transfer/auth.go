package transfer

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestConnection(config ConnectionConfig) error {
	client, err := connectSSH(config)
	if err != nil {
		return err
	}
	defer client.Close()

	return nil
}

func connectSSH(config ConnectionConfig) (*ssh.Client, error) {
	clientConfig, err := buildSSHClientConfig(config)
	if err != nil {
		return nil, err
	}

	client, err := ssh.Dial("tcp", config.Address(), clientConfig)
	if err != nil {
		return nil, fmt.Errorf("dial SSH server: %w", err)
	}

	return client, nil
}

func buildSSHClientConfig(config ConnectionConfig) (*ssh.ClientConfig, error) {
	authMethods, err := buildAuthMethods(config)
	if err != nil {
		return nil, err
	}

	hostKeyCallback, err := buildHostKeyCallback(config.Insecure)
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User:            config.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}, nil
}

func buildAuthMethods(config ConnectionConfig) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	if config.KeyPath != "" {
		signer, err := loadPrivateKey(config)
		if err != nil {
			return nil, err
		}

		methods = append(methods, ssh.PublicKeys(signer))
	}

	if config.Password != "" {
		methods = append(methods, ssh.Password(config.Password))
	}

	if envPassword := config.PasswordFromEnv(); envPassword != "" {
		methods = append(methods, ssh.Password(envPassword))
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no usable authentication method found")
	}

	return methods, nil
}

func loadPrivateKey(config ConnectionConfig) (ssh.Signer, error) {
	keyPath, err := config.ExpandedKeyPath()
	if err != nil {
		return nil, err
	}

	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key %q: %w", keyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("parse private key %q: %w", keyPath, err)
	}

	return signer, nil
}

func buildHostKeyCallback(insecure bool) (ssh.HostKeyCallback, error) {
	if insecure {
		return ssh.InsecureIgnoreHostKey(), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("find home directory: %w", err)
	}

	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")
	callback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("load known_hosts from %q: %w", knownHostsPath, err)
	}

	return callback, nil
}
