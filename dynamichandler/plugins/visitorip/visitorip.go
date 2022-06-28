package visitorip

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/RussellLuo/caddy-ext/dynamichandler/caddymiddleware"
)

// Middleware implements an HTTP handler that writes the
// visitor's IP address to a file or stream.
type Middleware struct {
	// The file or stream to write to. Can be "stdout"
	// or "stderr".
	Output string `json:"output,omitempty"`

	w io.Writer
}

func New() caddymiddleware.Middleware {
	return &Middleware{}
}

func (m *Middleware) Provision() error {
	switch m.Output {
	case "stdout":
		m.w = os.Stdout
	case "stderr":
		m.w = os.Stderr
	default:
		return fmt.Errorf("an output stream is required")
	}
	return nil
}

func (m *Middleware) Validate() error {
	if m.w == nil {
		return fmt.Errorf("no writer")
	}
	return nil
}

func (m *Middleware) Cleanup() error { return nil }

func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddymiddleware.Handler) error {
	_, _ = m.w.Write([]byte(r.RemoteAddr + "\n"))
	return next.ServeHTTP(w, r)
}
