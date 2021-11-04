package ratelimit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"go.uber.org/zap"
)

func TestRateLimit_ServeHTTP(t *testing.T) {
	next := caddyhttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusOK)
		return nil
	})

	cases := []struct {
		inRL            *RateLimit
		inReq           *http.Request
		wantStatusCodes []int
	}{
		{
			inRL: &RateLimit{
				Key:    "{query.id}",
				Rate:   "2r/m",
				logger: zap.NewNop(),
			},
			inReq: httptest.NewRequest(http.MethodGet, "/foo?id=1", nil),
			wantStatusCodes: []int{
				http.StatusOK,
				http.StatusOK,
				http.StatusTooManyRequests,
			},
		},
	}
	for _, c := range cases {
		_ = c.inRL.provision()
		//c.inRL.Cleanup()

		var gotStatusCodes []int
		for i := 0; i < len(c.wantStatusCodes); i++ {
			// Build the request object.
			repl := caddyhttp.NewTestReplacer(c.inReq)
			ctx := context.WithValue(c.inReq.Context(), caddy.ReplacerCtxKey, repl)
			req := c.inReq.WithContext(ctx)

			// Build the response object.
			w := httptest.NewRecorder()

			_ = c.inRL.ServeHTTP(w, req, next)

			// Collect the response status code.
			resp := w.Result()
			gotStatusCodes = append(gotStatusCodes, resp.StatusCode)
		}
		if !reflect.DeepEqual(gotStatusCodes, c.wantStatusCodes) {
			t.Fatalf("StatusCodes: got (%#v), want (%#v)", gotStatusCodes, c.wantStatusCodes)
		}
	}
}

func TestParseVar(t *testing.T) {
	cases := []struct {
		in         string
		want       *Var
		wantErrStr string
	}{
		{
			in: "{path.id}",
			want: &Var{
				Raw:  "{path.id}",
				Name: "{http.request.uri.path.id}",
			},
		},
		{
			in: "{query.id}",
			want: &Var{
				Raw:  "{query.id}",
				Name: "{http.request.uri.query.id}",
			},
		},
		{
			in: "{header.id}",
			want: &Var{
				Raw:  "{header.id}",
				Name: "{http.request.header.id}",
			},
		},
		{
			in: "{cookie.id}",
			want: &Var{
				Raw:  "{cookie.id}",
				Name: "{http.request.cookie.id}",
			},
		},
		{
			in: "{remote.host}",
			want: &Var{
				Raw:  "{remote.host}",
				Name: "{http.request.remote.host}",
			},
		},
		{
			in: "{remote.port}",
			want: &Var{
				Raw:  "{remote.port}",
				Name: "{http.request.remote.port}",
			},
		},
		{
			in: "{remote.ip}",
			want: &Var{
				Raw:  "{remote.ip}",
				Name: "{http.request.remote.ip}",
			},
		},
		{
			in: "{remote.host_prefix.24}",
			want: &Var{
				Raw:  "{remote.host_prefix.24}",
				Name: "{http.request.remote.host_prefix}",
				Bits: 24,
			},
		},
		{
			in: "{remote.ip_prefix.64}",
			want: &Var{
				Raw:  "{remote.ip_prefix.64}",
				Name: "{http.request.remote.ip_prefix}",
				Bits: 64,
			},
		},
		{
			in:         "{remote.host_prefix}",
			wantErrStr: `invalid key variable: "{remote.host_prefix}"`,
		},
		{
			in:         "{remote.ip_prefix.xx}",
			wantErrStr: `invalid key variable: "{remote.ip_prefix.xx}"`,
		},
		{
			in: "{http.request.uri.path.id}",
			want: &Var{
				Raw:  "{http.request.uri.path.id}",
				Name: "{http.request.uri.path.id}",
			},
		},
		{
			in:         "{unknown.id}",
			wantErrStr: `unrecognized key variable: "{unknown.id}"`,
		},
		{
			in:         "{unknown}",
			wantErrStr: `invalid key variable: "{unknown}"`,
		},
	}

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			v, err := ParseVar(c.in)
			if err != nil && err.Error() != c.wantErrStr {
				t.Fatalf("ErrStr: got (%#v), want (%#v)", err.Error(), c.wantErrStr)
			}
			if !reflect.DeepEqual(v, c.want) {
				t.Fatalf("Out: got (%#v), want (%#v)", v, c.want)
			}
		})
	}
}

func TestVar_Evaluate(t *testing.T) {
	cases := []struct {
		name       string
		inVar      *Var
		inReq      func() *http.Request
		wantValue  string
		wantErrStr string
	}{
		{
			name: "query",
			inVar: &Var{
				Name: "{http.request.uri.query.id}",
			},
			inReq: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/foo?id=1", nil)
			},
			wantValue: "1",
		},
		{
			name: "peer ip",
			inVar: &Var{
				Name: "{http.request.remote.host}",
			},
			inReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("X-Forwarded-For", "192.168.0.1, 192.168.0.2")
				return req
			},
			wantValue: "192.0.2.1",
		},
		{
			name: "peer port",
			inVar: &Var{
				Name: "{http.request.remote.port}",
			},
			inReq: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/", nil)
			},
			wantValue: "1234",
		},
		{
			name: "forwarded ip",
			inVar: &Var{
				Name: "{http.request.remote.ip}",
			},
			inReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set("X-Forwarded-For", "192.168.0.1, 192.168.0.2")
				return req
			},
			wantValue: "192.168.0.1",
		},
		{
			name: "ipv4 prefix",
			inVar: &Var{
				Name: "{http.request.remote.ip_prefix}",
				Bits: 24,
			},
			inReq: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/", nil)
			},
			wantValue: "192.0.2.0/24",
		},
		{
			name: "ipv6 prefix",
			inVar: &Var{
				Name: "{http.request.remote.ip_prefix}",
				Bits: 64,
			},
			inReq: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.RemoteAddr = "[2001:db8:85a3:8d3:1319:8a2e:370:7348]:443"
				return req
			},
			wantValue: "2001:db8:85a3:8d3::/64",
		},
		{
			name: "bad ipv4 prefix",
			inVar: &Var{
				Name: "{http.request.remote.ip_prefix}",
				Bits: 64,
			},
			inReq: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/", nil)
			},
			wantErrStr: "prefix length 64 too large for IPv4",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Build the request object.
			r := c.inReq()
			repl := caddyhttp.NewTestReplacer(r)
			ctx := context.WithValue(r.Context(), caddy.ReplacerCtxKey, repl)
			req := r.WithContext(ctx)

			value, err := c.inVar.Evaluate(req)
			if err != nil && err.Error() != c.wantErrStr {
				t.Fatalf("ErrStr: got (%#v), want (%#v)", err.Error(), c.wantErrStr)
			}
			if value != c.wantValue {
				t.Fatalf("Value: got (%#v), want (%#v)", value, c.wantValue)
			}
		})
	}
}
