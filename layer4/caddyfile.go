package layer4

import (
	"encoding/json"
	"strconv"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/mholt/caddy-l4/layer4"
	"github.com/mholt/caddy-l4/modules/l4echo"
	"github.com/mholt/caddy-l4/modules/l4proxy"
	"github.com/mholt/caddy-l4/modules/l4proxyprotocol"
)

func init() {
	httpcaddyfile.RegisterGlobalOption("layer4", parseLayer4)
}

// parseLayer4 sets up the "layer4" global option from Caddyfile tokens. Syntax:
//
//   layer4 {
//       <listens...> {
//           l4echo
//       }
//
//       <listens...> {
//           l4proxy [<options...>]
//       }
//   }
//
func parseLayer4(d *caddyfile.Dispenser, _ interface{}) (interface{}, error) {
	app := &layer4.App{Servers: make(map[string]*layer4.Server)}

	for d.Next() {
		for i := 0; d.NextBlock(0); i++ {
			server := new(layer4.Server)

			server.Listen = append(server.Listen, d.Val())
			for _, arg := range d.RemainingArgs() {
				server.Listen = append(server.Listen, arg)
			}

			for d.NextBlock(1) {
				switch d.Val() {
				case "l4echo", "echo":
					server.Routes = append(server.Routes, &layer4.Route{
						HandlersRaw: []json.RawMessage{caddyconfig.JSONModuleObject(new(l4echo.Handler), "handler", "echo", nil)},
					})

				case "proxy_protocol":
					handler, err := parseProxyProtocol(d)
					if err != nil {
						return nil, err
					}
					server.Routes = append(server.Routes, &layer4.Route{
						HandlersRaw: []json.RawMessage{caddyconfig.JSONModuleObject(handler, "handler", "proxy_protocol", nil)},
					})

				case "l4proxy", "proxy":
					handler, err := parseProxy(d)
					if err != nil {
						return nil, err
					}
					server.Routes = append(server.Routes, &layer4.Route{
						HandlersRaw: []json.RawMessage{caddyconfig.JSONModuleObject(handler, "handler", "proxy", nil)},
					})
				}
			}

			app.Servers["srv"+strconv.Itoa(i)] = server
		}
	}

	// tell Caddyfile adapter that this is the JSON for an app
	return httpcaddyfile.App{
		Name:  "layer4",
		Value: caddyconfig.JSON(app, nil),
	}, nil
}

// parseProxyProtocol sets up a "proxy_protocol" handler from Caddyfile tokens. Syntax:
//
//   proxy_protocol {
//       timeout <duration>
//       allow   <cidrs...>
//   }
//
func parseProxyProtocol(d *caddyfile.Dispenser) (*l4proxyprotocol.Handler, error) {
	h := new(l4proxyprotocol.Handler)

	// No same-line options are supported
	if len(d.RemainingArgs()) > 0 {
		return nil, d.ArgErr()
	}

	for nesting := d.Nesting(); d.NextBlock(nesting); {
		switch d.Val() {
		case "timeout":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			timeout, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return nil, d.Errf("parsing proxy_protocol timeout duration: %v", err)
			}
			h.Timeout = caddy.Duration(timeout)

		case "allow":
			args := d.RemainingArgs()
			if len(args) == 0 {
				return nil, d.ArgErr()
			}
			h.Allow = append(h.Allow, args...)

		default:
			return nil, d.ArgErr()
		}
	}

	return h, nil
}

// parseL4proxy sets up a "proxy" handler from Caddyfile tokens. Syntax:
//
//   proxy [<upstreams...>] {
//       # backends
//       to <upstreams...>
//   	 ...
//
//       # load balancing
//       lb_policy       <name> [<options...>]
//       lb_try_duration <duration>
//       lb_try_interval <interval>
//
//       # active health checking
//       health_port     <port>
//       health_interval <interval>
//       health_timeout  <duration>
//
//       # sending the PROXY protocol
//       proxy_protocol <version>
//   }
//
func parseProxy(d *caddyfile.Dispenser) (*l4proxy.Handler, error) {
	h := new(l4proxy.Handler)

	appendUpstream := func(addresses ...string) {
		for _, addr := range addresses {
			h.Upstreams = append(h.Upstreams, &l4proxy.Upstream{
				Dial: []string{addr},
			})
		}
	}

	appendUpstream(d.RemainingArgs()...)

	for nesting := d.Nesting(); d.NextBlock(nesting); {
		switch d.Val() {
		case "to":
			args := d.RemainingArgs()
			if len(args) == 0 {
				return nil, d.ArgErr()
			}
			appendUpstream(args...)

		case "lb_policy":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			if h.LoadBalancing != nil && h.LoadBalancing.SelectionPolicyRaw != nil {
				return nil, d.Err("load balancing selection policy already specified")
			}
			if h.LoadBalancing == nil {
				h.LoadBalancing = new(l4proxy.LoadBalancing)
			}

			name := d.Val()
			modID := "layer4.proxy.selection_policies." + name
			mod, err := UnmarshalL4proxySelectionModule(d, modID)
			if err != nil {
				return nil, err
			}

			sel, ok := mod.(l4proxy.Selector)
			if !ok {
				return nil, d.Errf("module %s (%T) is not a l4proxy.Selector", modID, mod)
			}
			h.LoadBalancing.SelectionPolicyRaw = caddyconfig.JSONModuleObject(sel, "policy", name, nil)

		case "lb_try_duration":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			if h.LoadBalancing == nil {
				h.LoadBalancing = new(l4proxy.LoadBalancing)
			}

			dur, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return nil, d.Errf("bad duration value %s: %v", d.Val(), err)
			}
			h.LoadBalancing.TryDuration = caddy.Duration(dur)

		case "lb_try_interval":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			if h.LoadBalancing == nil {
				h.LoadBalancing = new(l4proxy.LoadBalancing)
			}

			dur, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return nil, d.Errf("bad interval value '%s': %v", d.Val(), err)
			}
			h.LoadBalancing.TryInterval = caddy.Duration(dur)

		case "health_port":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			if h.HealthChecks == nil {
				h.HealthChecks = new(l4proxy.HealthChecks)
			}
			if h.HealthChecks.Active == nil {
				h.HealthChecks.Active = new(l4proxy.ActiveHealthChecks)
			}

			portNum, err := strconv.Atoi(d.Val())
			if err != nil {
				return nil, d.Errf("bad port number '%s': %v", d.Val(), err)
			}
			h.HealthChecks.Active.Port = portNum

		case "health_interval":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			if h.HealthChecks == nil {
				h.HealthChecks = new(l4proxy.HealthChecks)
			}
			if h.HealthChecks.Active == nil {
				h.HealthChecks.Active = new(l4proxy.ActiveHealthChecks)
			}

			dur, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return nil, d.Errf("bad interval value %s: %v", d.Val(), err)
			}
			h.HealthChecks.Active.Interval = caddy.Duration(dur)

		case "health_timeout":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			if h.HealthChecks == nil {
				h.HealthChecks = new(l4proxy.HealthChecks)
			}
			if h.HealthChecks.Active == nil {
				h.HealthChecks.Active = new(l4proxy.ActiveHealthChecks)
			}

			dur, err := caddy.ParseDuration(d.Val())
			if err != nil {
				return nil, d.Errf("bad timeout value %s: %v", d.Val(), err)
			}
			h.HealthChecks.Active.Timeout = caddy.Duration(dur)

		case "proxy_protocol":
			if !d.NextArg() {
				return nil, d.ArgErr()
			}
			h.ProxyProtocol = d.Val()
		}
	}

	return h, nil
}

// UnmarshalL4proxySelectionModule is like `caddyfile.UnmarshalModule`, but for
// l4proxy's selection modules, which do not implement `caddyfile.Unmarshaler` yet.
func UnmarshalL4proxySelectionModule(d *caddyfile.Dispenser, moduleID string) (caddy.Module, error) {
	mod, err := caddy.GetModule(moduleID)
	if err != nil {
		return nil, d.Errf("getting module named '%s': %v", moduleID, err)
	}
	inst := mod.New()

	if err = UnmarshalL4ProxySelectionCaddyfile(inst, d.NewFromNextSegment()); err != nil {
		return nil, err
	}
	return inst, nil
}

func UnmarshalL4ProxySelectionCaddyfile(inst caddy.Module, d *caddyfile.Dispenser) error {
	switch sel := inst.(type) {
	case *l4proxy.RandomSelection,
		*l4proxy.LeastConnSelection,
		*l4proxy.RoundRobinSelection,
		*l4proxy.FirstSelection,
		*l4proxy.IPHashSelection:

		for d.Next() {
			if d.NextArg() {
				return d.ArgErr()
			}
		}

	case *l4proxy.RandomChoiceSelection:
		for d.Next() {
			if !d.NextArg() {
				return d.ArgErr()
			}
			chooseStr := d.Val()
			choose, err := strconv.Atoi(chooseStr)
			if err != nil {
				return d.Errf("invalid choice value '%s': %v", chooseStr, err)
			}
			sel.Choose = choose
		}
	}

	return nil
}
