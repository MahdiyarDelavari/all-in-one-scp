package transfer

import (
	"fmt"
	"io"
	"path"

	"github.com/pkg/sftp"
)

// RemoteCopyRequest describes one server-to-server copy operation.
type RemoteCopyRequest struct {
	Source          ConnectionConfig
	Destination     ConnectionConfig
	SourcePath      string
	DestinationPath string
	Excludes        []string
}

func CopyRemoteToRemote(request RemoteCopyRequest, logger Logger) error {
	sourceSSH, err := connectSSH(request.Source)
	if err != nil {
		return fmt.Errorf("connect source server: %w", err)
	}
	defer sourceSSH.Close()

	destinationSSH, err := connectSSH(request.Destination)
	if err != nil {
		return fmt.Errorf("connect destination server: %w", err)
	}
	defer destinationSSH.Close()

	sourceSFTP, err := sftp.NewClient(sourceSSH)
	if err != nil {
		return fmt.Errorf("create source SFTP client: %w", err)
	}
	defer sourceSFTP.Close()

	destinationSFTP, err := sftp.NewClient(destinationSSH)
	if err != nil {
		return fmt.Errorf("create destination SFTP client: %w", err)
	}
	defer destinationSFTP.Close()

	sourcePath := cleanRemotePath(request.SourcePath)
	sourceInfo, err := sourceSFTP.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("stat source path %q: %w", sourcePath, err)
	}

	destinationRoot := cleanRemotePath(request.DestinationPath)
	if sourceInfo.IsDir() {
		destinationBase := path.Join(destinationRoot, path.Base(sourcePath))
		return copyRemoteDirectoryToRemote(sourceSFTP, destinationSFTP, sourcePath, destinationBase, request.Excludes, logger)
	}

	destinationFilePath := path.Join(destinationRoot, path.Base(sourcePath))
	return copyRemoteFileToRemote(sourceSFTP, destinationSFTP, sourcePath, destinationFilePath, logger)
}

func copyRemoteDirectoryToRemote(sourceClient *sftp.Client, destinationClient *sftp.Client, sourceDir string, destinationDir string, excludes []string, logger Logger) error {
	if shouldExclude(sourceDir, excludes) {
		logLine(logger, "Skipping directory: %s", sourceDir)
		return nil
	}

	if err := destinationClient.MkdirAll(destinationDir); err != nil {
		return fmt.Errorf("create destination directory %q: %w", destinationDir, err)
	}

	entries, err := sourceClient.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("read source directory %q: %w", sourceDir, err)
	}

	for _, entry := range entries {
		childSourcePath := path.Join(sourceDir, entry.Name())
		childDestinationPath := path.Join(destinationDir, entry.Name())

		if shouldExclude(childSourcePath, excludes) {
			logLine(logger, "Skipping: %s", childSourcePath)
			continue
		}

		if entry.IsDir() {
			if err := copyRemoteDirectoryToRemote(sourceClient, destinationClient, childSourcePath, childDestinationPath, excludes, logger); err != nil {
				return err
			}

			continue
		}

		if err := copyRemoteFileToRemote(sourceClient, destinationClient, childSourcePath, childDestinationPath, logger); err != nil {
			return err
		}
	}

	return nil
}

func copyRemoteFileToRemote(sourceClient *sftp.Client, destinationClient *sftp.Client, sourcePath string, destinationPath string, logger Logger) error {
	logLine(logger, "Copying: %s -> %s", sourcePath, destinationPath)

	sourceFile, err := sourceClient.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source file %q: %w", sourcePath, err)
	}
	defer sourceFile.Close()

	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("stat source file %q: %w", sourcePath, err)
	}

	if err := destinationClient.MkdirAll(path.Dir(destinationPath)); err != nil {
		return fmt.Errorf("create destination parent directory for %q: %w", destinationPath, err)
	}

	destinationFile, err := destinationClient.Create(destinationPath)
	if err != nil {
		return fmt.Errorf("create destination file %q: %w", destinationPath, err)
	}

	if _, err := io.Copy(destinationFile, sourceFile); err != nil {
		destinationFile.Close()
		return fmt.Errorf("copy %q to %q: %w", sourcePath, destinationPath, err)
	}

	if err := destinationFile.Close(); err != nil {
		return fmt.Errorf("close destination file %q: %w", destinationPath, err)
	}

	if err := destinationClient.Chmod(destinationPath, sourceInfo.Mode().Perm()); err != nil {
		logLine(logger, "Warning: could not copy permissions for %s: %v", destinationPath, err)
	}

	return nil
}
