package server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tsukinoko-kun/orca/config"
	"github.com/tsukinoko-kun/orca/public"
)

type Server struct {
	httpServer        *http.Server
	opencodeServers   map[string]string
	opencodeServersMu sync.RWMutex
}

func New() *Server {
	s := &Server{
		opencodeServers: make(map[string]string),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/api/", s.handleAPIProxy)
	if frontendDevURL, ok := os.LookupEnv("FRONTEND_DEV_URL"); ok {
		mux.HandleFunc("/", s.handleDevProxy(frontendDevURL))
	} else {
		mux.HandleFunc("/", s.handleStaticFiles)
	}
	s.httpServer = &http.Server{
		Addr:         config.Address,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}
	return s
}

func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) handleDevProxy(frontendURL string) http.HandlerFunc {
	target, err := url.Parse(frontendURL)
	if err != nil {
		log.Fatalf("invalid FRONTEND_DEV_URL: %v", err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") == "websocket" {
			s.proxyWebSocket(w, r, target)
			return
		}

		proxyURL := *target
		proxyURL.Path = r.URL.Path
		proxyURL.RawQuery = r.URL.RawQuery

		proxyReq, err := http.NewRequest(r.Method, proxyURL.String(), r.Body)
		if err != nil {
			http.Error(w, "proxy error", http.StatusInternalServerError)
			return
		}

		for key, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(proxyReq)
		if err != nil {
			http.Error(w, "proxy error", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

func (s *Server) proxyWebSocket(w http.ResponseWriter, r *http.Request, target *url.URL) {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "webserver doesn't support hijacking", http.StatusInternalServerError)
		return
	}

	targetConn, err := net.Dial("tcp", target.Host)
	if err != nil {
		http.Error(w, "failed to connect to backend", http.StatusBadGateway)
		return
	}
	defer targetConn.Close()

	req := &http.Request{
		Method: "GET",
		URL:    &url.URL{Path: r.URL.Path, RawQuery: r.URL.RawQuery},
		Header: r.Header,
		Host:   target.Host,
	}

	if err := req.Write(targetConn); err != nil {
		http.Error(w, "failed to write request", http.StatusInternalServerError)
		return
	}

	clientConn, bufrw, err := hj.Hijack()
	if err != nil {
		http.Error(w, "failed to hijack connection", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	targetReader := bufio.NewReader(targetConn)
	resp, err := http.ReadResponse(targetReader, req)
	if err != nil {
		return
	}

	if err := resp.Write(bufrw); err != nil {
		return
	}
	if err := bufrw.Flush(); err != nil {
		return
	}

	done := make(chan struct{}, 2)

	go func() {
		io.Copy(targetConn, clientConn)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(clientConn, targetConn)
		done <- struct{}{}
	}()

	<-done
}

func (s *Server) handleStaticFiles(w http.ResponseWriter, r *http.Request) {
	fsPath := strings.TrimPrefix(r.URL.Path, "/")

	_, err := fs.Stat(public.Public, fsPath)
	if os.IsNotExist(err) {
		w.Header().Set("Content-Type", "text/html")
		http.ServeFileFS(w, r, public.Public, "index.html")
		return
	}

	if strings.HasPrefix(r.URL.Path, "/assets/") {
		w.Header().Set("Cache-Control", "public, max-age=2592000") // 30 days
	}

	switch ext := filepath.Ext(fsPath); ext {
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	case ".mp4":
		w.Header().Set("Content-Type", "video/mp4")
	case ".webm":
		w.Header().Set("Content-Type", "video/webm")
	default:
		fmt.Printf("unknown file extension: %s\n", ext)
	}

	http.ServeFileFS(w, r, public.Public, fsPath)
}

func (s *Server) handleAPIProxy(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/"), "/", 2)
	if len(pathParts) < 1 || pathParts[0] == "" {
		http.Error(w, "missing sessionID in path", http.StatusBadRequest)
		return
	}

	sessionID := pathParts[0]
	apiPath := ""
	if len(pathParts) > 1 {
		apiPath = "/" + pathParts[1]
	}

	s.opencodeServersMu.RLock()
	targetURL, ok := s.opencodeServers[sessionID]
	s.opencodeServersMu.RUnlock()

	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	proxyURL := targetURL + apiPath
	if r.URL.RawQuery != "" {
		proxyURL += "?" + r.URL.RawQuery
	}

	proxyReq, err := http.NewRequest(r.Method, proxyURL, r.Body)
	if err != nil {
		http.Error(w, "proxy error", http.StatusInternalServerError)
		return
	}

	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, "proxy error", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
