package handlers

import "net/http"

type AuthHandlers struct{}

func (h AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("hello world"))
}

func (h AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("hello world"))
}
