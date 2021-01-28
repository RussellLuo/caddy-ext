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
		inReqPath       string
		wantStatusCodes []int
	}{
		{
			inRL: &RateLimit{
				Key:    "{query.id}",
				Rate:   "2r/m",
				logger: zap.NewNop(),
			},
			inReqPath: "/foo?id=1",
			wantStatusCodes: []int{
				http.StatusOK,
				http.StatusOK,
				http.StatusTooManyRequests,
			},
		},
	}
	for _, c := range cases {
		c.inRL.provision()
		//c.inRL.Cleanup()

		var gotStatusCodes []int
		for i := 0; i < len(c.wantStatusCodes); i++ {
			// Build the request object.
			req := httptest.NewRequest(http.MethodGet, c.inReqPath, nil)
			repl := caddyhttp.NewTestReplacer(req)
			ctx := context.WithValue(req.Context(), caddy.ReplacerCtxKey, repl)
			req = req.WithContext(ctx)

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
		want       string
		wantErrStr string
	}{
		{
			in:   "{path.id}",
			want: "{http.request.uri.path.id}",
		},
		{
			in:   "{query.id}",
			want: "{http.request.uri.query.id}",
		},
		{
			in:   "{header.id}",
			want: "{http.request.header.id}",
		},
		{
			in:   "{cookie.id}",
			want: "{http.request.cookie.id}",
		},
		{
			in:   "{remote.ip}",
			want: "{http.request.remote.ip}",
		},
		{
			in:   "{http.request.uri.path.id}",
			want: "{http.request.uri.path.id}",
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
		v, err := parseVar(c.in)
		if err != nil && err.Error() != c.wantErrStr {
			t.Fatalf("ErrStr: got (%#v), want (%#v)", err.Error(), c.wantErrStr)
		}
		if v != c.want {
			t.Fatalf("Out: got (%#v), want (%#v)", v, c.want)
		}
	}
}
