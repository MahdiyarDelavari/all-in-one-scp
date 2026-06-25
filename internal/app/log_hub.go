package app

import "sync"

type logHub struct {
	mu      sync.Mutex
	nextID  int
	clients map[int]chan string
}

func newLogHub() *logHub {
	return &logHub{
		clients: make(map[int]chan string),
	}
}

func (h *logHub) subscribe() (<-chan string, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	id := h.nextID
	h.nextID++

	ch := make(chan string, 32)
	h.clients[id] = ch

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		if existing, ok := h.clients[id]; ok {
			delete(h.clients, id)
			close(existing)
		}
	}
}

func (h *logHub) broadcast(line string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, ch := range h.clients {
		select {
		case ch <- line:
		default:
		}
	}
}
