package dynamichandler

import (
	"encoding/json"
	"fmt"
	"go/build"
	"net/http"
	"path"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"github.com/traefik/yaegi/stdlib/unsafe"
	"go.uber.org/zap"

	"github.com/RussellLuo/caddy-ext/dynamichandler/caddymiddleware"
	"github.com/RussellLuo/caddy-ext/dynamichandler/yaegisymbols"
)

func init() {
	caddy.RegisterModule(DynamicHandler{})
}

// DynamicHandler implements a handler that can execute plugins (written in Go) dynamically.
type DynamicHandler struct {
	// The path to the plugin code in Go.
	Path string `json:"path,omitempty"`
	// The plugin configuration in JSON format.
	Config string `json:"config,omitempty"`

	middleware caddymiddleware.Middleware
	logger     *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (DynamicHandler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.dynamic_handler",
		New: func() caddy.Module { return new(DynamicHandler) },
	}
}

// Provision implements caddy.Provisioner.
func (dh *DynamicHandler) Provision(ctx caddy.Context) error {
	dh.logger = ctx.Logger(dh)
	return dh.provision()
}

func (dh *DynamicHandler) provision() error {
	m, err := dh.eval()
	if err != nil {
		return err
	}

	dh.middleware = m
	return dh.middleware.Provision()
}

// Validate implements caddy.Validator.
func (dh *DynamicHandler) Validate() error {
	if dh.Path == "" {
		return fmt.Errorf("empty path")
	}
	return dh.middleware.Validate()
}

// Cleanup implements caddy.CleanerUpper.
func (dh *DynamicHandler) Cleanup() error {
	if dh.middleware != nil {
		return dh.middleware.Cleanup()
	}
	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (dh *DynamicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	return dh.middleware.ServeHTTP(w, r, next)
}

func (dh *DynamicHandler) eval() (caddymiddleware.Middleware, error) {
	i := interp.New(interp.Options{GoPath: build.Default.GOPATH})
	for _, exports := range []interp.Exports{
		stdlib.Symbols,
		unsafe.Symbols,
		yaegisymbols.Symbols,
	} {
		if err := i.Use(exports); err != nil {
			return nil, err
		}
	}

	if _, err := i.EvalPath(dh.Path); err != nil {
		return nil, err
	}

	basePkg := strings.ReplaceAll(path.Base(path.Dir(dh.Path)), "-", "_")
	newFunc, err := i.Eval(basePkg + ".New")
	if err != nil {
		return nil, err
	}

	newMiddleware, ok := newFunc.Interface().(func() caddymiddleware.Middleware)
	if !ok {
		return nil, fmt.Errorf("%s.New does not implement `func() caddymiddleware.Middleware`", basePkg)
	}
	middleware := newMiddleware()

	if len(dh.Config) > 0 {
		out := middleware.(yaegisymbols.Middleware).IValue
		if err := json.Unmarshal([]byte(dh.Config), out); err != nil {
			return nil, err
		}
	}

	return middleware, nil
}

// Interface guards
var (
	_ caddy.Provisioner           = (*DynamicHandler)(nil)
	_ caddy.Validator             = (*DynamicHandler)(nil)
	_ caddy.CleanerUpper          = (*DynamicHandler)(nil)
	_ caddyhttp.MiddlewareHandler = (*DynamicHandler)(nil)
)
