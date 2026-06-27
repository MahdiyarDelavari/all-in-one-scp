package app

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"sync"

	"allinonescp/internal/transfer"
)

//go:embed static
var staticFiles embed.FS

type Server struct {
	logs     *logHub
	sshLogs  *logHub
	sshShell *transfer.BrowserShell
	quit     chan struct{}
	once     sync.Once
}

type connectionPayload struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	User        string `json:"user"`
	KeyPath     string `json:"keyPath"`
	Password    string `json:"password"`
	PasswordEnv string `json:"passwordEnv"`
	Insecure    bool   `json:"insecure"`
}

type requestPayload struct {
	Source          connectionPayload `json:"source"`
	Destination     connectionPayload `json:"destination"`
	RemotePath      string            `json:"remotePath"`
	LocalPath       string            `json:"localPath"`
	DestinationPath string            `json:"destinationPath"`
	Excludes        string            `json:"excludes"`
	TransferMode    string            `json:"transferMode"`
}

type apiResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

type sshInputPayload struct {
	Input string `json:"input"`
}

func New() *Server {
	return &Server{
		logs:     newLogHub(),
		sshLogs:  newLogHub(),
		sshShell: transfer.NewBrowserShell(),
		quit:     make(chan struct{}),
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	frontend, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(err)
	}

	mux.Handle("/", http.FileServer(http.FS(frontend)))
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/test", s.handleTestConnection)
	mux.HandleFunc("/api/ssh/events", s.handleSSHEvents)
	mux.HandleFunc("/api/ssh/connect", s.handleSSHConnect)
	mux.HandleFunc("/api/ssh/input", s.handleSSHInput)
	mux.HandleFunc("/api/ssh/disconnect", s.handleSSHDisconnect)
	mux.HandleFunc("/api/download", s.handleDownload)
	mux.HandleFunc("/api/remote-copy", s.handleRemoteCopy)
	mux.HandleFunc("/api/quit", s.handleQuit)

	return mux
}

func (s *Server) Quit() <-chan struct{} {
	return s.quit
}

func (s *Server) Log(line string) {
	s.logs.broadcast(line)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	s.handleStream(w, r, s.logs, "Connected to activity log.")
}

func (s *Server) handleSSHEvents(w http.ResponseWriter, r *http.Request) {
	s.handleStream(w, r, s.sshLogs, "Connected to SSH output.")
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request, hub *logHub, firstMessage string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	client, unsubscribe := hub.subscribe()
	defer unsubscribe()

	fmt.Fprintf(w, "data: %s\n\n", escapeSSE(firstMessage))
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case line, ok := <-client:
			if !ok {
				return
			}

			fmt.Fprintf(w, "data: %s\n\n", escapeSSE(line))
			flusher.Flush()
		}
	}
}

func (s *Server) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	payload, ok := s.decodePayload(w, r)
	if !ok {
		return
	}

	config, err := payload.sourceConfig()
	if err != nil {
		s.writeOperationError(w, err)
		return
	}

	s.Log("Testing source connection to " + config.Address())
	if err := transfer.TestConnection(config); err != nil {
		s.Log("Source connection failed: " + err.Error())
		s.writeOperationError(w, err)
		return
	}

	if !payload.isRemoteCopyMode() {
		s.Log("Source connection succeeded.")
		writeJSON(w, http.StatusOK, apiResponse{OK: true, Message: "Connection succeeded."})
		return
	}

	destinationConfig, err := payload.destinationConfig()
	if err != nil {
		s.writeOperationError(w, err)
		return
	}

	s.Log("Testing destination connection to " + destinationConfig.Address())
	if err := transfer.TestConnection(destinationConfig); err != nil {
		s.Log("Destination connection failed: " + err.Error())
		s.writeOperationError(w, err)
		return
	}

	s.Log("Source and destination connections succeeded.")
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Message: "Both server connections succeeded."})
}

func (s *Server) handleSSHConnect(w http.ResponseWriter, r *http.Request) {
	payload, ok := s.decodePayload(w, r)
	if !ok {
		return
	}

	config, err := payload.sourceConfig()
	if err != nil {
		s.writeOperationError(w, err)
		return
	}

	s.Log("Starting in-page SSH session for " + config.Address())
	s.sshLogs.broadcast("")
	if err := s.sshShell.Start(config, func(chunk string) {
		s.sshLogs.broadcast(chunk)
	}); err != nil {
		s.Log("Could not start SSH session: " + err.Error())
		s.writeOperationError(w, err)
		return
	}

	s.sshLogs.broadcast("[Connected to " + config.Address() + "]\n")
	s.Log("SSH session connected.")
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Message: "SSH connected."})
}

func (s *Server) handleSSHInput(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload sshInputPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(payload.Input) == "" {
		s.writeOperationError(w, fmt.Errorf("command cannot be empty"))
		return
	}

	if err := s.sshShell.SendLine(payload.Input); err != nil {
		s.writeOperationError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{OK: true, Message: "Command sent."})
}

func (s *Server) handleSSHDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := s.sshShell.Close(); err != nil {
		s.writeOperationError(w, err)
		return
	}

	s.sshLogs.broadcast("\n[SSH session disconnected]\n")
	s.Log("SSH session disconnected.")
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Message: "SSH disconnected."})
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	payload, ok := s.decodePayload(w, r)
	if !ok {
		return
	}

	request, err := payload.downloadRequest()
	if err != nil {
		s.writeOperationError(w, err)
		return
	}

	s.Log("Starting download from " + request.RemotePath)
	err = transfer.PullFromRemote(request, func(line string) {
		s.Log(line)
	})
	if err != nil {
		s.Log("Download failed: " + err.Error())
		s.writeOperationError(w, err)
		return
	}

	s.Log("Download finished.")
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Message: "Download finished."})
}

func (s *Server) handleRemoteCopy(w http.ResponseWriter, r *http.Request) {
	payload, ok := s.decodePayload(w, r)
	if !ok {
		return
	}

	request, err := payload.remoteCopyRequest()
	if err != nil {
		s.writeOperationError(w, err)
		return
	}

	s.Log("Starting server-to-server copy from " + request.SourcePath + " to " + request.DestinationPath)
	err = transfer.CopyRemoteToRemote(request, func(line string) {
		s.Log(line)
	})
	if err != nil {
		s.Log("Server-to-server copy failed: " + err.Error())
		s.writeOperationError(w, err)
		return
	}

	s.Log("Server-to-server copy finished.")
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Message: "Server-to-server copy finished."})
}

func (s *Server) handleQuit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.Log("Shutting down app.")
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Message: "App is shutting down."})
	s.once.Do(func() {
		close(s.quit)
	})
}

func (s *Server) decodePayload(w http.ResponseWriter, r *http.Request) (requestPayload, bool) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return requestPayload{}, false
	}

	var payload requestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return requestPayload{}, false
	}

	return payload, true
}

func (s *Server) writeOperationError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusOK, apiResponse{
		OK:    false,
		Error: err.Error(),
	})
}

func (p connectionPayload) connectionConfig(label string) (transfer.ConnectionConfig, error) {
	config := transfer.ConnectionConfig{
		Host:        strings.TrimSpace(p.Host),
		Port:        p.Port,
		User:        strings.TrimSpace(p.User),
		KeyPath:     strings.TrimSpace(p.KeyPath),
		Password:    p.Password,
		PasswordEnv: strings.TrimSpace(p.PasswordEnv),
		Insecure:    p.Insecure,
	}

	if err := config.Validate(); err != nil {
		return transfer.ConnectionConfig{}, fmt.Errorf("%s: %w", label, err)
	}

	return config, nil
}

func (p requestPayload) sourceConfig() (transfer.ConnectionConfig, error) {
	return p.Source.connectionConfig("source server")
}

func (p requestPayload) destinationConfig() (transfer.ConnectionConfig, error) {
	return p.Destination.connectionConfig("destination server")
}

func (p requestPayload) downloadRequest() (transfer.DownloadRequest, error) {
	config, err := p.sourceConfig()
	if err != nil {
		return transfer.DownloadRequest{}, err
	}

	request := transfer.DownloadRequest{
		Connection: config,
		RemotePath: strings.TrimSpace(p.RemotePath),
		LocalPath:  strings.TrimSpace(p.LocalPath),
		Excludes:   splitCommaList(p.Excludes),
	}

	if request.RemotePath == "" {
		return transfer.DownloadRequest{}, fmt.Errorf("remote path is required")
	}

	if request.LocalPath == "" {
		return transfer.DownloadRequest{}, fmt.Errorf("local path is required")
	}

	return request, nil
}

func (p requestPayload) remoteCopyRequest() (transfer.RemoteCopyRequest, error) {
	sourceConfig, err := p.sourceConfig()
	if err != nil {
		return transfer.RemoteCopyRequest{}, err
	}

	destinationConfig, err := p.destinationConfig()
	if err != nil {
		return transfer.RemoteCopyRequest{}, err
	}

	request := transfer.RemoteCopyRequest{
		Source:          sourceConfig,
		Destination:     destinationConfig,
		SourcePath:      strings.TrimSpace(p.RemotePath),
		DestinationPath: strings.TrimSpace(p.DestinationPath),
		Excludes:        splitCommaList(p.Excludes),
	}

	if request.SourcePath == "" {
		return transfer.RemoteCopyRequest{}, fmt.Errorf("source path is required")
	}

	if request.DestinationPath == "" {
		return transfer.RemoteCopyRequest{}, fmt.Errorf("destination path is required")
	}

	return request, nil
}

func (p requestPayload) isRemoteCopyMode() bool {
	return strings.EqualFold(strings.TrimSpace(p.TransferMode), "remoteCopy")
}

func splitCommaList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			items = append(items, trimmed)
		}
	}

	return items
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func escapeSSE(line string) string {
	return strings.ReplaceAll(line, "\n", " ")
}
