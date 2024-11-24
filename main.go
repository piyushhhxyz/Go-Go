package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// ShortUrl represents a shortened URL with its original URL and creation time.
type ShortUrl struct {
	ShortUrl    string
	OriginalUrl string
	CreatedAt   time.Time
}

// urlDB is a concurrent-safe map to store ShortUrl data.
var urlDB = struct {
	sync.RWMutex
	m map[string]ShortUrl
}{m: make(map[string]ShortUrl)}

func mainw() {
	router := mux.NewRouter()

	router.HandleFunc("/", HealthCheck).Methods("GET")
	router.HandleFunc("/shorten", ShortUrlHandler).Methods("GET")
	router.HandleFunc("/{shorten_url}", RedirectUrlHandler).Methods("GET")

	log.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

// HealthCheck provides a simple health check endpoint.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Server is running babyy")
	http.Redirect(w, r, "https://www.youtube.com", http.StatusSeeOther)
	json.NewEncoder(w).Encode(map[string]string{"message": "Server is running"})
}

// generateShortUrl creates a short URL from the original URL and stores it in the concurrent map.
func generateShortUrl(originalUrl string) string {
	hasher := md5.New()
	hasher.Write([]byte(originalUrl))
	data := hasher.Sum(nil)
	shortUrl := hex.EncodeToString(data)[:6]

	urlDB.Lock()
	urlDB.m[shortUrl] = ShortUrl{
		ShortUrl:    shortUrl,
		OriginalUrl: originalUrl,
		CreatedAt:   time.Now(),
	}
	urlDB.Unlock()

	return shortUrl
}

// ShortUrlHandler handles the creation of a short URL.
func ShortUrlHandler(w http.ResponseWriter, r *http.Request) {
	originalUrl := r.URL.Query().Get("url")
	shortUrl := generateShortUrl(originalUrl)

	urlDB.RLock()
	getUrlFromDB := urlDB.m[shortUrl]
	urlDB.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"short_url":    getUrlFromDB.ShortUrl,
		"original_url": getUrlFromDB.OriginalUrl,
	})
}

// RedirectUrlHandler handles the redirection from a short URL to the original URL.
func RedirectUrlHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortUrl := vars["shorten_url"]

	urlDB.RLock()
	shortUrlData, exists := urlDB.m[shortUrl]
	urlDB.RUnlock()

	if !exists {
		log.Printf("Short URL '%s' not found", shortUrl)
		http.Error(w, "Short URL not found", http.StatusNotFound)
		return
	}

	log.Printf("Redirecting short URL '%s' to original URL '%s'", shortUrl, shortUrlData.OriginalUrl)
	http.Redirect(w, r, shortUrlData.OriginalUrl, http.StatusSeeOther)
}