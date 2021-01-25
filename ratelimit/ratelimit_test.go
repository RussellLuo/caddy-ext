package ratelimit

import (
	"testing"
)

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
