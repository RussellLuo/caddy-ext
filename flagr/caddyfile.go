package flagr

import (
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("flagr", parseCaddyfile)
}

// parseCaddyfile sets up a handler for flagr from Caddyfile tokens.
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	f := new(Flagr)
	if err := f.UnmarshalCaddyfile(h.Dispenser); err != nil {
		return nil, err
	}
	return f, nil
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler. Syntax:
//
// flagr <url> {
//     evaluator <evaluator> [<refresh_interval>]
//     entity_id <entity_id>
//     entity_context {
//         <key1>    <value1>
//         <key2>    <value2>
//         ...
//     }
//     flag_keys <key1> <key2> ...
//     bind_variant_keys_to <bind_variant_keys_to>
// }
//
func (f *Flagr) UnmarshalCaddyfile(d *caddyfile.Dispenser) (err error) {
	for d.Next() {
		if !d.NextArg() {
			return d.ArgErr()
		}
		f.URL = d.Val()

		for nesting := d.Nesting(); d.NextBlock(nesting); {
			switch d.Val() {
			case "evaluator":
				if d.NextArg() {
					f.Evaluator = d.Val()
				}
				if f.Evaluator == "local" {
					if d.NextArg() {
						f.RefreshInterval = d.Val()
					}
				}

			case "entity_id":
				if !d.NextArg() {
					return d.ArgErr()
				}
				f.EntityID = d.Val()

			case "entity_context":
				f.EntityContext = make(map[string]interface{})
				for nesting := d.Nesting(); d.NextBlock(nesting); {
					key := d.Val()
					if !d.NextArg() {
						return d.ArgErr()
					}
					value := d.Val()

					if key == "" || value == "" {
						return d.Err("empty key or empty value within entity_context")
					}
					f.EntityContext[key] = value
				}

			case "flag_keys":
				for d.NextArg() {
					f.FlagKeys = append(f.FlagKeys, d.Val())
				}
				if len(f.FlagKeys) == 0 {
					return d.ArgErr()
				}

			case "bind_variant_keys_to":
				if !d.NextArg() {
					return d.ArgErr()
				}
				f.BindVariantKeysTo = d.Val()
			}
		}
	}

	return nil
}

// Interface guards
var (
	_ caddyfile.Unmarshaler = (*Flagr)(nil)
)
