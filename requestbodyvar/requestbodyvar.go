package requestbodyvar

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/tidwall/gjson"
)

const (
	reqBodyReplPrefix = "http.request.body."
)

func init() {
	caddy.RegisterModule(RequestBodyVar{})
	httpcaddyfile.RegisterHandlerDirective("request_body_var", parseCaddyfile)
}

// RequestBodyVar implements an HTTP handler that replaces {http.request.body.*}
// with the value of the given field from request body, if any.
type RequestBodyVar struct{}

// CaddyModule returns the Caddy module information.
func (RequestBodyVar) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.request_body_var",
		New: func() caddy.Module { return new(RequestBodyVar) },
	}
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (m RequestBodyVar) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)

	bodyVars := func(key string) (interface{}, bool) {
		if r == nil || r.Body == nil {
			return nil, false
		}
		// Close the real body since we will replace it with a fake one.
		defer r.Body.Close()

		// Copy the request body.
		buf := new(bytes.Buffer)
		_, err := io.Copy(buf, r.Body)
		if err != nil {
			return nil, false
		}

		// Replace the real body with buffered data.
		r.Body = ioutil.NopCloser(buf)

		// Get the value of the given field from the body.
		value := gjson.Get(buf.String(), key[len(reqBodyReplPrefix):])
		return value.String(), true
	}

	repl.Map(bodyVars)

	return next.ServeHTTP(w, r)
}

// UnmarshalCaddyfile - this is a no-op
func (m *RequestBodyVar) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	return nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	m := new(RequestBodyVar)
	err := m.UnmarshalCaddyfile(h.Dispenser)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// Interface guards
var (
	_ caddyhttp.MiddlewareHandler = (*RequestBodyVar)(nil)
	_ caddyfile.Unmarshaler       = (*RequestBodyVar)(nil)
)
