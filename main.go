package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type KeyStore struct {
	mu   sync.Mutex
	data map[string]StoredKey
}

type StoredKey struct {
	Key       string
	ExpiresAt time.Time
}

var store = KeyStore{
	data: make(map[string]StoredKey),
}

// Генерация случайного токена
func generateToken() string {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// Отправка ключа
func sendKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key string `json:"key"`
	}

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil || body.Key == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	token := generateToken()

	store.mu.Lock()
	store.data[token] = StoredKey{
		Key:       body.Key,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	store.mu.Unlock()

	json.NewEncoder(w).Encode(map[string]string{
		"token": token,
	})
}

// Получение ключа
func getKey(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	store.mu.Lock()
	entry, exists := store.data[token]
	if exists {
		delete(store.data, token) // удаляем после получения
	}
	store.mu.Unlock()

	if !exists {
		http.Error(w, "not found or already used", http.StatusNotFound)
		return
	}

	if time.Now().After(entry.ExpiresAt) {
		http.Error(w, "expired", http.StatusGone)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"key": entry.Key,
	})
}

// Очистка просроченных ключей
func cleanup() {
	for {
		time.Sleep(1 * time.Minute)

		store.mu.Lock()
		for k, v := range store.data {
			if time.Now().After(v.ExpiresAt) {
				delete(store.data, k)
			}
		}
		store.mu.Unlock()
	}
}

func main() {
	go cleanup()

	http.HandleFunc("/send", sendKey)
	http.HandleFunc("/get", getKey)

	log.Println("Server running on https://localhost:8443")

	err := http.ListenAndServeTLS("0.0.0.0:8443", "cert.pem", "key.pem", nil)
	if err != nil {
		log.Fatal(err)
	}
}
