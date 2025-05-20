package main

import (
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

type StatusResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type TemplateData struct {
	Status    string
	Timestamp string
}

type RateLimiter struct {
	mu      sync.Mutex
	clients map[string]*clientData
}

type clientData struct {
	tokens      int
	lastRequest time.Time
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*clientData),
	}
}

func (rl *RateLimiter) Allow(ip string, limit int, refillRate time.Duration) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	client, exists := rl.clients[ip]
	if !exists {
		rl.clients[ip] = &clientData{tokens: limit - 1, lastRequest: now}
		return true
	}

	elapsed := now.Sub(client.lastRequest)
	refilled := int(elapsed / refillRate)
	if refilled > 0 {
		client.tokens = min(limit, client.tokens+refilled)
		client.lastRequest = now
	}

	if client.tokens <= 0 {
		return false
	}

	client.tokens--
	return true
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8081"
		log.Printf("BACKEND_URL not set. Using default: %s", backendURL)
	}

	tmpl, err := template.ParseFiles("templates/index.html.tmpl")
	if err != nil {
		log.Fatalf("Failed to parse template: %v", err)
	}

	rateLimiter := NewRateLimiter()
	requestLimit := 60
	refillRate := 60 * time.Second

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		if !rateLimiter.Allow(ip, requestLimit, refillRate) {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		status := StatusResponse{
			Status:    "Unavailable",
			Timestamp: time.Now().Format(time.RFC3339),
		}

		resp, err := http.Get(backendURL + "/api/status")
		if err == nil && resp.StatusCode == 200 {
			defer resp.Body.Close()
			json.NewDecoder(resp.Body).Decode(&status)
		}

		_ = tmpl.Execute(w, TemplateData{
			Status:    status.Status,
			Timestamp: status.Timestamp,
		})
	})

	// Proxy Swagger UI from backend
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyResp, err := http.Get(backendURL + "/swagger/" + r.URL.Path)
		if err != nil {
			http.Error(w, "Failed to proxy to backend", http.StatusBadGateway)
			return
		}
		defer proxyResp.Body.Close()

		for k, v := range proxyResp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(proxyResp.StatusCode)
		io.Copy(w, proxyResp.Body)
	})))

	logged := loggingMiddleware(mux)

	port := "8080"
	log.Printf("Frontend service running on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, logged))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rr, r)

		duration := time.Since(start)
		clientIP := r.RemoteAddr
		method := r.Method
		path := r.URL.Path

		log.Printf("%s %s %d [%s] (%s)", method, path, rr.statusCode, clientIP, duration)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}
