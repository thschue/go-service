package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

type StatusResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

func main() {
	connStr := os.Getenv("DATABASE_URL")

	if connStr != "" {
		var err error
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			log.Printf("Could not open DB: %v", err)
		} else if err = db.Ping(); err != nil {
			log.Printf("Could not connect to DB: %v", err)
		} else {
			log.Println("Connected to the database successfully")
		}
	} else {
		log.Println("DATABASE_URL not set — running in degraded mode")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", statusHandler)
	mux.HandleFunc("/api/openapi.yaml", openapiHandler)
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.Dir("static"))))

	logged := loggingMiddleware(mux)

	port := "8081"
	log.Printf("Backend running on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, logged))
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	status := "Disconnected"
	if db != nil {
		if err := db.Ping(); err == nil {
			status = "Connected"
		}
	}

	resp := StatusResponse{
		Status:    status,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func openapiHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-yaml")
	http.ServeFile(w, r, "openapi.yaml")
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
