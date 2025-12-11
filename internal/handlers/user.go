package handlers

import "net/http"

type UserHandler struct{}

func (u UserHandler) Get(w http.ResponseWriter, r *http.Request) {}

func (u UserHandler) Post(w http.ResponseWriter, r *http.Request) {}
