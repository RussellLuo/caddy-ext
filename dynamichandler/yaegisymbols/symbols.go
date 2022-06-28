package yaegisymbols

import (
	"reflect"
)

// Symbols stores the map of caddymiddleware package symbols.
var Symbols = map[string]map[string]reflect.Value{}

type Middleware = _github_com_RussellLuo_caddy_ext_dynamichandler_caddymiddleware_Middleware

//go:generate yaegi extract github.com/RussellLuo/caddy-ext/dynamichandler/caddymiddleware
