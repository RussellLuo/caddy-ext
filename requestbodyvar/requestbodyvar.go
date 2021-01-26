package requestbodyvar

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

const (
	reqBodyReplPrefix = "http.request.body."

	// For the request's buffered JSON body
	jsonBufCtxKey caddy.CtxKey = "json_buf"
)

func init() {
	caddy.RegisterModule(RequestBodyVar{})
	httpcaddyfile.RegisterHandlerDirective("request_body_var", parseCaddyfile)
}

// RequestBodyVar implements an HTTP handler that replaces {http.request.body.*}
// with the value of the given field from request body, if any.
type RequestBodyVar struct {
	logger *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (RequestBodyVar) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.request_body_var",
		New: func() caddy.Module { return new(RequestBodyVar) },
	}
}

// Provision implements caddy.Provisioner.
func (rbv *RequestBodyVar) Provision(ctx caddy.Context) (err error) {
	rbv.logger = ctx.Logger(rbv)
	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (rbv RequestBodyVar) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	bodyVars := func(key string) (interface{}, bool) {
		if !strings.HasPrefix(key, reqBodyReplPrefix) {
			return nil, false
		}

		// First of all, try to get the value from the buffered JSON body, if any.
		buf, ok := r.Context().Value(jsonBufCtxKey).(*bytes.Buffer)
		if ok {
			rbv.logger.Debug("got from the buffer", zap.String("key", key))
			return getJSONField(buf, key), true
		}

		rbv.logger.Debug("got from the body", zap.String("key", key))

		// Otherwise, try to get the value by reading the request body.
		if r == nil || r.Body == nil {
			return "", true
		}
		// Close the real body since we will replace it with a fake one.
		defer r.Body.Close()

		// TODO: Throw an error for non-JSON body?

		// Copy the request body.
		buf = new(bytes.Buffer)
		_, err := io.Copy(buf, r.Body)
		if err != nil {
			return "", true
		}

		// Replace the real body with buffered data.
		r.Body = ioutil.NopCloser(buf)

		// Add the buffered JSON body into the context for the request.
		ctx := context.WithValue(r.Context(), jsonBufCtxKey, buf)
		r = r.WithContext(ctx)

		return getJSONField(buf, key), true
	}

	repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
	repl.Map(bodyVars)

	return next.ServeHTTP(w, r)
}

// getJSONField gets the value of the given field from the JSON body,
// which is buffered in buf.
func getJSONField(buf *bytes.Buffer, key string) string {
	if buf == nil {
		return ""
	}
	value := gjson.Get(buf.String(), key[len(reqBodyReplPrefix):])
	return value.String()
}

// UnmarshalCaddyfile - this is a no-op
func (rbv *RequestBodyVar) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	return nil
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	rbv := new(RequestBodyVar)
	err := rbv.UnmarshalCaddyfile(h.Dispenser)
	if err != nil {
		return nil, err
	}

	return rbv, nil
}

// Interface guards
var (
	_ caddyhttp.MiddlewareHandler = (*RequestBodyVar)(nil)
	_ caddyfile.Unmarshaler       = (*RequestBodyVar)(nil)
)
