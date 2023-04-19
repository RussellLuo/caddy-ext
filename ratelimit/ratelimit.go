package ratelimit

import (
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

var (
	regexpFullVar  = regexp.MustCompile(`^\{http\.request\..+\}$`)
	regexpShortVar = regexp.MustCompile(`^\{(\w+)\.(.+)\}$`)
	// "host_prefix.<bits>" or "ip_prefix.<bits>"
	regexpPrefixVar = regexp.MustCompile(`^(host_prefix|ip_prefix)\.([0-9]+)$`)
	regexpRate      = regexp.MustCompile(`^(\d+)r/(s|m)$`)
)

func init() {
	caddy.RegisterModule(RateLimit{})
}

// RateLimit implements a handler for rate-limiting.
//
// If a client exceeds the rate limit, an HTTP error with status `<reject_status>` will
// be returned. This error can be handled using the conventional error handlers.
// See [handle_errors](https://caddyserver.com/docs/caddyfile/directives/handle_errors)
// for how to set up error handlers.
type RateLimit struct {
	// The variable used to differentiate one client from another.
	//
	// Currently supported variables:
	//
	// - `{path.<var>}`
	// - `{query.<var>}`
	// - `{header.<VAR>}`
	// - `{cookie.<var>}`
	// - `{body.<var>}` (requires the [requestbodyvar](https://github.com/RussellLuo/caddy-ext/tree/master/requestbodyvar) extension)
	// - `{remote.host}` (ignores the `X-Forwarded-For` header)
	// - `{remote.port}`
	// - `{remote.ip}` (prefers the first IP in the `X-Forwarded-For` header)
	// - `{remote.host_prefix.<bits>}` (CIDR block version of `{remote.host}`)
	// - `{remote.ip_prefix.<bits>}` (CIDR block version of `{remote.ip}`)
	Key string `json:"key,omitempty"`

	// The request rate limit (per key value) specified in requests
	// per second (r/s) or requests per minute (r/m).
	Rate string `json:"rate,omitempty"`

	// The size (i.e. the number of key values) of the LRU zone that
	// keeps states of these key values. Defaults to 10,000.
	ZoneSize int `json:"zone_size,omitempty"`

	// The HTTP status code of the response when a client exceeds the rate.
	// Defaults to 429 (Too Many Requests).
	RejectStatusCode int `json:"reject_status,omitempty"`

	keyVar *Var
	zone   *Zone

	logger *zap.Logger
}

// CaddyModule returns the Caddy module information.
func (RateLimit) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.rate_limit",
		New: func() caddy.Module { return new(RateLimit) },
	}
}

// Provision implements caddy.Provisioner.
func (rl *RateLimit) Provision(ctx caddy.Context) (err error) {
	rl.logger = ctx.Logger(rl)
	return rl.provision()
}

func (rl *RateLimit) provision() (err error) {
	rl.keyVar, err = ParseVar(rl.Key)
	if err != nil {
		return err
	}

	rateSize, rateLimit, err := parseRate(rl.Rate)
	if err != nil {
		return err
	}

	if rl.ZoneSize == 0 {
		rl.ZoneSize = 10000 // At most 10,000 keys by default
	}

	rl.zone, err = NewZone(rl.ZoneSize, rateSize, int64(rateLimit))
	if err != nil {
		return err
	}

	if rl.RejectStatusCode == 0 {
		rl.RejectStatusCode = http.StatusTooManyRequests
	}

	return nil
}

// Cleanup cleans up the resources made by rl during provisioning.
func (rl *RateLimit) Cleanup() error {
	if rl.zone != nil {
		rl.zone.Purge()
	}
	return nil
}

// Validate implements caddy.Validator.
func (rl *RateLimit) Validate() error {
	if rl.keyVar == nil {
		return fmt.Errorf("no key variable")
	}
	if rl.zone == nil {
		return fmt.Errorf("no zone created")
	}
	if http.StatusText(rl.RejectStatusCode) == "" {
		return fmt.Errorf("unknown code reject_status: %d", rl.RejectStatusCode)
	}
	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (rl *RateLimit) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	keyValue, err := rl.keyVar.Evaluate(r)
	if err != nil {
		rl.logger.Error("failed to evaluate variable",
			zap.String("variable", rl.keyVar.Raw),
			zap.Error(err),
		)
		return next.ServeHTTP(w, r)
	}

	w.Header().Add("RateLimit-Policy", rl.zone.RateLimitPolicyHeader())

	if keyValue != "" && !rl.zone.Allow(keyValue) {
		rl.logger.Debug("request is rejected",
			zap.String("variable", rl.keyVar.Raw),
			zap.String("value", keyValue),
		)

		w.WriteHeader(rl.RejectStatusCode)
		// Return an error to invoke possible error handlers.
		return caddyhttp.Error(rl.RejectStatusCode, nil)
	}

	return next.ServeHTTP(w, r)
}

type Var struct {
	Raw  string
	Name string
	Bits int
}

// ParseVar transforms shorthand variables into Caddy-style placeholders.
//
// Examples for shorthand variables:
//
// - `{path.<var>}`
// - `{query.<var>}`
// - `{header.<VAR>}`
// - `{cookie.<var>}`
// - `{body.<var>}`
// - `{remote.host}`
// - `{remote.port}`
// - `{remote.ip}`
// - `{remote.host_prefix.<bits>}`
// - `{remote.ip_prefix.<bits>}`
func ParseVar(s string) (*Var, error) {
	v := &Var{Raw: s}
	if regexpFullVar.MatchString(s) {
		// If the variable is already a fully-qualified Caddy placeholder,
		// return it as is.
		v.Name = s
		return v, nil
	}

	result := regexpShortVar.FindStringSubmatch(s)
	if len(result) != 3 {
		return nil, fmt.Errorf("invalid key variable: %q", s)
	}
	location, name := result[1], result[2]

	switch location {
	case "path":
		v.Name = fmt.Sprintf("{http.request.uri.path.%s}", name)
	case "query":
		v.Name = fmt.Sprintf("{http.request.uri.query.%s}", name)
	case "header":
		v.Name = fmt.Sprintf("{http.request.header.%s}", name)
	case "cookie":
		v.Name = fmt.Sprintf("{http.request.cookie.%s}", name)
	case "body":
		v.Name = fmt.Sprintf("{http.request.body.%s}", name)
	case "remote":
		if name == "host" || name == "port" || name == "ip" {
			v.Name = fmt.Sprintf("{http.request.remote.%s}", name)
			return v, nil
		}

		r := regexpPrefixVar.FindStringSubmatch(name)
		if len(r) != 3 {
			return nil, fmt.Errorf("invalid key variable: %q", s)
		}

		v.Name = fmt.Sprintf("{http.request.remote.%s}", r[1])

		if r[2] == "" {
			return nil, fmt.Errorf("invalid key variable: %q", s)
		}
		bits, err := strconv.Atoi(r[2])
		if err != nil {
			return nil, err
		}
		v.Bits = bits
	default:
		return nil, fmt.Errorf("unrecognized key variable: %q", s)
	}

	return v, nil
}

func (v *Var) Evaluate(r *http.Request) (value string, err error) {
	switch v.Name {
	case "{http.request.remote.ip}":
		ip, err := getClientIP(r, true)
		if err != nil {
			return "", err
		}
		return ip.String(), nil
	case "{http.request.remote.host_prefix}":
		return v.evaluatePrefix(r, false)
	case "{http.request.remote.ip_prefix}":
		return v.evaluatePrefix(r, true)
	default:
		repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
		value = repl.ReplaceAll(v.Name, "")
		return value, nil
	}
}

func (v *Var) evaluatePrefix(r *http.Request, forwarded bool) (value string, err error) {
	ip, err := getClientIP(r, forwarded)
	if err != nil {
		return "", err
	}
	prefix, err := ip.Prefix(v.Bits)
	if err != nil {
		return "", err
	}
	return prefix.Masked().String(), nil
}

func getClientIP(r *http.Request, forwarded bool) (netip.Addr, error) {
	remote := r.RemoteAddr
	if forwarded {
		if fwdFor := r.Header.Get("X-Forwarded-For"); fwdFor != "" {
			remote = strings.TrimSpace(strings.Split(fwdFor, ",")[0])
		}
	}
	ipStr, _, err := net.SplitHostPort(remote)
	if err != nil {
		ipStr = remote // OK; probably didn't have a port
	}
	return netip.ParseAddr(ipStr)
}

func parseRate(rate string) (size time.Duration, limit int, err error) {
	if rate == "" {
		return 0, 0, fmt.Errorf("missing rate")
	}

	result := regexpRate.FindStringSubmatch(rate)
	if len(result) != 3 {
		return 0, 0, fmt.Errorf("invalid rate: %s", rate)
	}
	limitStr, sizeStr := result[1], result[2]

	switch sizeStr {
	case "s":
		size = time.Second
	case "m":
		size = time.Minute
	}

	limit, err = strconv.Atoi(limitStr)
	if err != nil {
		return 0, 0, fmt.Errorf("size-limit must be an integer; invalid: %v", err)
	}

	return
}

// Interface guards
var (
	_ caddy.Provisioner           = (*RateLimit)(nil)
	_ caddy.CleanerUpper          = (*RateLimit)(nil)
	_ caddy.Validator             = (*RateLimit)(nil)
	_ caddyhttp.MiddlewareHandler = (*RateLimit)(nil)
)
