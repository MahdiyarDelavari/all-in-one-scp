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
	logs *logHub
	quit chan struct{}
	once sync.Once
}

type requestPayload struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	User        string `json:"user"`
	KeyPath     string `json:"keyPath"`
	Password    string `json:"password"`
	PasswordEnv string `json:"passwordEnv"`
	Insecure    bool   `json:"insecure"`
	RemotePath  string `json:"remotePath"`
	LocalPath   string `json:"localPath"`
	Excludes    string `json:"excludes"`
}

type apiResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func New() *Server {
	return &Server{
		logs: newLogHub(),
		quit: make(chan struct{}),
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
	mux.HandleFunc("/api/open-ssh", s.handleOpenSSH)
	mux.HandleFunc("/api/download", s.handleDownload)
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
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	client, unsubscribe := s.logs.subscribe()
	defer unsubscribe()

	fmt.Fprintf(w, "data: %s\n\n", escapeSSE("Connected to activity log."))
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

	config, err := payload.connectionConfig()
	if err != nil {
		s.writeOperationError(w, err)
		return
	}

	s.Log("Testing connection to " + config.Address())
	if err := transfer.TestConnection(config); err != nil {
		s.Log("Connection failed: " + err.Error())
		s.writeOperationError(w, err)
		return
	}

	s.Log("Connection succeeded.")
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Message: "Connection succeeded."})
}

func (s *Server) handleOpenSSH(w http.ResponseWriter, r *http.Request) {
	payload, ok := s.decodePayload(w, r)
	if !ok {
		return
	}

	config, err := payload.connectionConfig()
	if err != nil {
		s.writeOperationError(w, err)
		return
	}

	s.Log("Opening SSH window for " + config.Address())
	if err := transfer.OpenInteractiveShell(config); err != nil {
		s.Log("Could not open SSH window: " + err.Error())
		s.writeOperationError(w, err)
		return
	}

	s.Log("SSH terminal launched.")
	writeJSON(w, http.StatusOK, apiResponse{OK: true, Message: "SSH terminal launched."})
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

func (p requestPayload) connectionConfig() (transfer.ConnectionConfig, error) {
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
		return transfer.ConnectionConfig{}, err
	}

	return config, nil
}

func (p requestPayload) downloadRequest() (transfer.DownloadRequest, error) {
	config, err := p.connectionConfig()
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
