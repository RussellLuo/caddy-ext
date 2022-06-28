package dynamichandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/RussellLuo/caddy-ext/dynamichandler/caddymiddleware"
)

func TestDynamic_eval(t *testing.T) {
	d := &DynamicHandler{
		Path:   "./plugins/visitorip/visitorip.go",
		Config: `{"output": "stdout"}`,
		logger: zap.NewNop(),
	}
	if err := d.provision(); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	next := caddymiddleware.HandlerFunc(func(http.ResponseWriter, *http.Request) error { return nil })
	if err := d.middleware.ServeHTTP(w, r, next); err != nil {
		t.Fatal(err)
	}
}
