package transfer

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
)

// DownloadRequest describes one remote copy operation.
type DownloadRequest struct {
	Connection ConnectionConfig
	RemotePath string
	LocalPath  string
	Excludes   []string
}

type Logger func(string)

func PullFromRemote(request DownloadRequest, logger Logger) error {
	client, err := connectSSH(request.Connection)
	if err != nil {
		return err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("create SFTP client: %w", err)
	}
	defer sftpClient.Close()

	remotePath := cleanRemotePath(request.RemotePath)
	remoteInfo, err := sftpClient.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("stat remote path %q: %w", remotePath, err)
	}

	localRoot, err := filepath.Abs(request.LocalPath)
	if err != nil {
		return fmt.Errorf("resolve local path %q: %w", request.LocalPath, err)
	}

	if remoteInfo.IsDir() {
		localBase := filepath.Join(localRoot, path.Base(remotePath))
		return copyRemoteDirectory(sftpClient, remotePath, localBase, request.Excludes, logger)
	}

	return copyRemoteFile(sftpClient, remotePath, filepath.Join(localRoot, path.Base(remotePath)), logger)
}

func copyRemoteDirectory(client *sftp.Client, remoteDir string, localDir string, excludes []string, logger Logger) error {
	if shouldExclude(remoteDir, excludes) {
		logLine(logger, "Skipping directory: %s", remoteDir)
		return nil
	}

	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return fmt.Errorf("create local directory %q: %w", localDir, err)
	}

	entries, err := client.ReadDir(remoteDir)
	if err != nil {
		return fmt.Errorf("read remote directory %q: %w", remoteDir, err)
	}

	for _, entry := range entries {
		childRemotePath := path.Join(remoteDir, entry.Name())
		childLocalPath := filepath.Join(localDir, entry.Name())

		if shouldExclude(childRemotePath, excludes) {
			logLine(logger, "Skipping: %s", childRemotePath)
			continue
		}

		if entry.IsDir() {
			if err := copyRemoteDirectory(client, childRemotePath, childLocalPath, excludes, logger); err != nil {
				return err
			}

			continue
		}

		if err := copyRemoteFile(client, childRemotePath, childLocalPath, logger); err != nil {
			return err
		}
	}

	return nil
}

func copyRemoteFile(client *sftp.Client, remotePath string, localPath string, logger Logger) error {
	logLine(logger, "Downloading: %s -> %s", remotePath, localPath)

	remoteFile, err := client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("open remote file %q: %w", remotePath, err)
	}
	defer remoteFile.Close()

	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return fmt.Errorf("create local parent directory for %q: %w", localPath, err)
	}

	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local file %q: %w", localPath, err)
	}
	defer localFile.Close()

	if _, err := io.Copy(localFile, remoteFile); err != nil {
		return fmt.Errorf("copy %q to %q: %w", remotePath, localPath, err)
	}

	return nil
}

func shouldExclude(remotePath string, excludes []string) bool {
	remotePath = cleanRemotePath(remotePath)
	name := path.Base(remotePath)

	for _, pattern := range excludes {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		if pattern == name || pattern == remotePath {
			return true
		}

		if matchesGlob(pattern, name) || matchesGlob(pattern, remotePath) {
			return true
		}
	}

	return false
}

func matchesGlob(pattern string, value string) bool {
	matched, err := path.Match(pattern, value)
	return err == nil && matched
}

func cleanRemotePath(remotePath string) string {
	trimmed := strings.TrimSpace(strings.ReplaceAll(remotePath, "\\", "/"))
	if trimmed == "" {
		return "."
	}

	return path.Clean(trimmed)
}

func logLine(logger Logger, format string, values ...any) {
	if logger == nil {
		return
	}

	logger(fmt.Sprintf(format, values...))
}
