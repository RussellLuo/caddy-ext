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

// parseCaddyfile sets up a handler from Caddyfile tokens.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	dh := new(DynamicHandler)
	if err := dh.UnmarshalCaddyfile(h.Dispenser); err != nil {
		return nil, err
	}
	return dh, nil
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler. Syntax:
//
// dynamic_handler <name> {
//     root <root>
//     config <config>
// }
//
func (dh *DynamicHandler) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	if !d.Next() {
		return d.ArgErr()
	}

	if !d.NextArg() {
		return d.ArgErr()
	}
	dh.Name = d.Val()

	// Get the path of the Caddyfile.
	caddyfilePath, err := filepath.Abs(d.File())
	if err != nil {
		return fmt.Errorf("failed to get absolute path of file: %s: %v", d.File(), err)
	}
	caddyfileDir := filepath.Dir(caddyfilePath)
	dh.Root = caddyfileDir // Defaults to the directory of the Caddyfile.

	for nesting := d.Nesting(); d.NextBlock(nesting); {
		switch d.Val() {
		case "root":
			if !d.NextArg() {
				return d.ArgErr()
			}
			root := d.Val()

			if filepath.IsAbs(root) {
				dh.Root = root
				return nil
			}

			// Make the path relative to the Caddyfile rather than the
			// current working directory.
			dh.Root = filepath.Join(caddyfileDir, root)

		case "config":
			if !d.NextArg() {
				return d.ArgErr()
			}
			dh.Config = d.Val()
		}
	}

	return nil
}

// Interface guards
var (
	_ caddyfile.Unmarshaler = (*DynamicHandler)(nil)
)
