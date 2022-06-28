package caddymiddleware

import (
	"net/http"
)

type Handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request) error
}

type HandlerFunc func(http.ResponseWriter, *http.Request) error

func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) error {
	return f(w, r)
}

type Middleware interface {
	Provision() error
	Validate() error
	Cleanup() error
	ServeHTTP(http.ResponseWriter, *http.Request, Handler) error
}
