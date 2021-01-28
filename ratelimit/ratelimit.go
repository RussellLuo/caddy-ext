package ratelimit

import (
	"fmt"
	"net"
	"net/http"
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
	regexpRate     = regexp.MustCompile(`^(\d+)r/(s|m)$`)
)

func init() {
	caddy.RegisterModule(RateLimit{})
}

// RateLimit implements a handler for rate-limiting.
type RateLimit struct {
	Key              string `json:"key,omitempty"`
	Rate             string `json:"rate,omitempty"`
	ZoneSize         int    `json:"zone_size,omitempty"`
	RejectStatusCode int    `json:"reject_status,omitempty"`

	keyVar string
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
	rl.keyVar, err = parseVar(rl.Key)
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
	if rl.keyVar == "" {
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
	keyValue, err := getVarValue(r, rl.keyVar)
	if err != nil {
		rl.logger.Error("failed to evaluate variable",
			zap.String("variable", rl.keyVar),
			zap.Error(err),
		)
		return next.ServeHTTP(w, r)
	}

	if keyValue != "" && !rl.zone.Allow(keyValue) {
		rl.logger.Debug("request is rejected",
			zap.String("variable", rl.keyVar),
			zap.String("value", keyValue),
		)

		w.WriteHeader(rl.RejectStatusCode)
		return nil
	}

	return next.ServeHTTP(w, r)
}

// parseVar transforms shorthand variables into Caddy-style placeholders.
//
// Examples for shorthand variables:
//
//     {path.<var>}
//     {query.<var>}
//     {header.<VAR>}
//     {cookie.<var>}
//     {body.<var>}
//     {remote.host}
//     {remote.port}
//     {remote.ip}
//
func parseVar(s string) (v string, err error) {
	if regexpFullVar.MatchString(s) {
		// If the variable is already a fully-qualified Caddy placeholder,
		// return it as is.
		return s, nil
	}

	result := regexpShortVar.FindStringSubmatch(s)
	if len(result) != 3 {
		return "", fmt.Errorf("invalid key variable: %q", s)
	}
	location, name := result[1], result[2]

	switch location {
	case "path":
		v = fmt.Sprintf("{http.request.uri.path.%s}", name)
	case "query":
		v = fmt.Sprintf("{http.request.uri.query.%s}", name)
	case "header":
		v = fmt.Sprintf("{http.request.header.%s}", name)
	case "cookie":
		v = fmt.Sprintf("{http.request.cookie.%s}", name)
	case "body":
		v = fmt.Sprintf("{http.request.body.%s}", name)
	case "remote":
		v = fmt.Sprintf("{http.request.remote.%s}", name)
	default:
		err = fmt.Errorf("unrecognized key variable: %q", s)
	}

	return
}

func getVarValue(r *http.Request, name string) (v string, err error) {
	switch name {
	case "{http.request.remote.ip}":
		var ip net.IP
		ip, err = getClientIP(r)
		if err != nil {
			return "", err
		}
		v = ip.String()
	default:
		repl := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
		v = repl.ReplaceAll(name, "")
	}
	return
}

// This function is borrowed from Caddy.
// See https://github.com/caddyserver/caddy/blob/3366384d9347447632ac334ffbbe35fb18738b90/modules/caddyhttp/matchers.go#L844-L860
func getClientIP(r *http.Request) (net.IP, error) {
	var remote string
	if fwdFor := r.Header.Get("X-Forwarded-For"); fwdFor != "" {
		remote = strings.TrimSpace(strings.Split(fwdFor, ",")[0])
	}
	if remote == "" {
		remote = r.RemoteAddr
	}

	ipStr, _, err := net.SplitHostPort(remote)
	if err != nil {
		ipStr = remote // OK; probably didn't have a port
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid client IP address: %s", ipStr)
	}

	return ip, nil
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
