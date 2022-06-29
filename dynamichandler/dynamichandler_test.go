package dynamichandler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/RussellLuo/caddy-ext/dynamichandler/caddymiddleware"
)

func TestDynamicHandler_eval(t *testing.T) {
	dh := &DynamicHandler{
		Name:   "visitorip",
		Root:   "plugins",
		Config: `{"output": "stdout"}`,
		logger: zap.NewNop(),
	}
	if err := dh.provision(); err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	next := caddymiddleware.HandlerFunc(func(http.ResponseWriter, *http.Request) error { return nil })
	if err := dh.middleware.ServeHTTP(w, r, next); err != nil {
		t.Fatal(err)
	}
}
