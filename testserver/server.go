package testserver

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
)

// TestServer provides a simple HTTP server for testing
type TestServer struct {
	server   *http.Server
	port     int
	listener net.Listener
}

// NewTestServer creates a new test server on the specified port
func NewTestServer(port int) *TestServer {
	mux := http.NewServeMux()
	
	// Status endpoint - returns the specified status code
	mux.HandleFunc("/status/", func(w http.ResponseWriter, r *http.Request) {
		// Extract status code from path /status/200, /status/404, etc.
		statusStr := r.URL.Path[len("/status/"):]
		if statusCode, err := strconv.Atoi(statusStr); err == nil {
			w.WriteHeader(statusCode)
			response := map[string]interface{}{
				"status": statusCode,
				"path":   r.URL.Path,
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
				return
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			if err := json.NewEncoder(w).Encode(map[string]string{"error": "invalid status code"}); err != nil {
				http.Error(w, "Failed to encode error response", http.StatusInternalServerError)
			}
		}
	})
	
	// GET endpoint - similar to httpbin.org/get
	mux.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		
		response := map[string]interface{}{
			"method": r.Method,
			"url":    r.URL.String(),
			"headers": r.Header,
			"origin": r.RemoteAddr,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	})
	
	// POST endpoint - similar to httpbin.org/post
	mux.HandleFunc("/post", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		
		var body interface{}
		if r.Header.Get("Content-Type") == "application/json" {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				// Ignore decode errors for malformed JSON
				body = nil
			}
		}
		
		response := map[string]interface{}{
			"method":  r.Method,
			"url":     r.URL.String(),
			"headers": r.Header,
			"origin":  r.RemoteAddr,
			"json":    body,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
	})
	
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	
	return &TestServer{
		server: server,
		port:   port,
	}
}

// Start starts the test server in the background
func (ts *TestServer) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", ts.port))
	if err != nil {
		return err
	}
	ts.listener = listener
	ts.port = listener.Addr().(*net.TCPAddr).Port
	
	go func() {
		if err := ts.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			_ = err // Explicitly ignore error
		}
	}()
	return nil
}

// Stop stops the test server
func (ts *TestServer) Stop() error {
	if ts.listener != nil {
		return ts.listener.Close()
	}
	return ts.server.Close()
}

// URL returns the base URL of the test server
func (ts *TestServer) URL() string {
	return fmt.Sprintf("http://localhost:%d", ts.port)
}