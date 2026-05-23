// Package serve runs the dev HTTP server with SSE-driven live reload.
package serve

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Server wraps an HTTP file server with a small SSE hub for live reload.
type Server struct {
	Dir  string // dist/ directory to serve
	Addr string // e.g. ":4321"

	hub *hub
	srv *http.Server
}

// New constructs a Server.
func New(dir, addr string) *Server {
	return &Server{Dir: dir, Addr: addr, hub: newHub()}
}

// Reload broadcasts a reload event to every connected browser tab.
func (s *Server) Reload() { s.hub.broadcast() }

// Run starts the HTTP server and blocks until ctx is cancelled.
func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/_ssg/reload", s.hub)
	mux.Handle("/", &devHandler{dir: s.Dir})

	s.srv = &http.Server{
		Addr:    s.Addr,
		Handler: logRequests(mux),
	}

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.srv.Shutdown(shutCtx)
	}()

	log.Printf("ssg: serving %s on http://localhost%s", s.Dir, s.Addr)
	err := s.srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// devHandler serves files from Dir.  For .html responses (including
// directory-index requests) it injects a small SSE client at end-of-body
// so file changes trigger an automatic reload.
type devHandler struct {
	dir string
}

func (h *devHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	// Prevent directory traversal.
	urlPath = filepath.Clean("/" + urlPath)
	fsPath := filepath.Join(h.dir, urlPath)

	info, err := os.Stat(fsPath)
	if err == nil && info.IsDir() {
		fsPath = filepath.Join(fsPath, "index.html")
		info, err = os.Stat(fsPath)
	}
	if err != nil {
		// Try /404.html for a nicer dev experience.
		if data, err := os.ReadFile(filepath.Join(h.dir, "404.html")); err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(injectReloadClient(data))
			return
		}
		http.NotFound(w, r)
		return
	}

	if strings.HasSuffix(fsPath, ".html") {
		data, err := os.ReadFile(fsPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(injectReloadClient(data))
		return
	}

	http.ServeFile(w, r, fsPath)
	_ = info
}

// injectReloadClient appends an SSE client script just before </body>.
// Falls back to appending at end-of-document if no </body> is found.
func injectReloadClient(data []byte) []byte {
	const snippet = `<script>(function(){const es=new EventSource("/_ssg/reload");es.addEventListener("reload",()=>location.reload());es.onerror=()=>setTimeout(()=>location.reload(),500);})();</script>`
	if i := bytes.LastIndex(data, []byte("</body>")); i >= 0 {
		out := make([]byte, 0, len(data)+len(snippet))
		out = append(out, data[:i]...)
		out = append(out, snippet...)
		out = append(out, data[i:]...)
		return out
	}
	return append(data, []byte(snippet)...)
}

// hub fans a single broadcast() call out to every connected SSE client.
type hub struct {
	mu      sync.Mutex
	clients map[chan struct{}]struct{}
}

func newHub() *hub { return &hub{clients: map[chan struct{}]struct{}{}} }

func (h *hub) subscribe() chan struct{} {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *hub) unsubscribe(ch chan struct{}) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
	close(ch)
}

func (h *hub) broadcast() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}

// ServeHTTP implements an SSE endpoint that emits a "reload" event each time
// broadcast() is called.
func (h *hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ch := h.subscribe()
	defer h.unsubscribe(ch)

	// Heartbeat keeps the connection alive through proxies / dev tools.
	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			fmt.Fprintf(w, "event: reload\ndata: {}\n\n")
			flusher.Flush()
		case <-tick.C:
			fmt.Fprintf(w, ":\n\n")
			flusher.Flush()
		}
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		// Skip SSE noise.
		if r.URL.Path != "/_ssg/reload" {
			log.Printf("[%s] %s (%s)", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
		}
	})
}
