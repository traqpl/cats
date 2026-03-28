package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//go:embed web
var webFS embed.FS

var store *ScoreStore
var nickRe = regexp.MustCompile(`^[A-Za-z0-9]{3}$`)

func main() {
	store = NewScoreStore(os.Getenv("DB_PATH"))

	_ = mime.AddExtensionType(".wasm", "application/wasm")
	_ = mime.AddExtensionType(".ogg", "audio/ogg")
	_ = mime.AddExtensionType(".flac", "audio/flac")
	_ = mime.AddExtensionType(".wav", "audio/wav")
	_ = mime.AddExtensionType(".mp3", "audio/mpeg")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8071"
	}

	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/scores", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			handleGetScores(w, r)
		case http.MethodPost:
			handlePostScore(w, r)
		default:
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})
	mux.Handle("/", http.FileServer(http.FS(sub)))

	addr := ":" + port
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("PURR & CARE → http://localhost%s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func handleGetScores(w http.ResponseWriter, r *http.Request) {
	n := 5
	if nStr := r.URL.Query().Get("n"); nStr != "" {
		if v, err := strconv.Atoi(nStr); err == nil && v > 0 && v <= 50 {
			n = v
		}
	}
	entries := store.Top(n)
	if entries == nil {
		entries = []ScoreEntry{}
	}
	_ = json.NewEncoder(w).Encode(entries)
}

func handlePostScore(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Nick  string `json:"nick"`
		Score int    `json:"score"`
		Days  int    `json:"days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}

	nick := strings.ToUpper(strings.TrimSpace(req.Nick))
	if !nickRe.MatchString(nick) {
		http.Error(w, `{"error":"nick must be 3 letters or digits"}`, http.StatusBadRequest)
		return
	}
	if req.Score < 0 || req.Score > 9999999 {
		http.Error(w, `{"error":"score out of range"}`, http.StatusBadRequest)
		return
	}
	if req.Days < 1 || req.Days > 7 {
		http.Error(w, `{"error":"invalid days"}`, http.StatusBadRequest)
		return
	}

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	entry := ScoreEntry{Nick: nick, Score: req.Score, Days: req.Days}
	msg, status := store.Add(entry, ip)
	if msg != "" {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
		return
	}
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte(`{"ok":true}`))
}
