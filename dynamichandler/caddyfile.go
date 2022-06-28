package dynamichandler

import (
	"fmt"
	"path/filepath"

	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("dynamic_handler", parseCaddyfile)
}

// parseCaddyfile sets up a handler for flagr from Caddyfile tokens.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	dh := new(DynamicHandler)
	if err := dh.UnmarshalCaddyfile(h.Dispenser); err != nil {
		return nil, err
	}
	return dh, nil
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler. Syntax:
//
// dynamic_handler <path> [config]
//
func (dh *DynamicHandler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	if d.Next() {
		args := d.RemainingArgs()
		switch len(args) {
		case 2:
			dh.Config = args[1]
			fallthrough

		case 1:
			path := args[0]
			if filepath.IsAbs(path) {
				dh.Path = path
				return nil
			}

			// Make the path relative to the current Caddyfile rather than the
			// current working directory.
			absFile, err := filepath.Abs(d.File())
			if err != nil {
				return fmt.Errorf("failed to get absolute path of file: %s: %v", d.File(), err)
			}
			dh.Path = filepath.Join(filepath.Dir(absFile), path)

		default:
			return d.ArgErr()
		}
	}
	return nil
}

// Interface guards
var (
	_ caddyfile.Unmarshaler = (*DynamicHandler)(nil)
)
