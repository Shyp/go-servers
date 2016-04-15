package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shyp/go-servers/test"
)

func ExampleRegexpHandler() {
	h := new(RegexpHandler)
	route := BuildRoute(`^/v1/jobs/(?P<Id>[^\s\/]+)$`)
	h.HandleFunc(route, []string{"GET", "POST"}, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})
}

func ExamplePprofMiddleware() {
	// Exposes /internal, /internal/trace, /internal/symbol, /internal/cmdline,
	// /internal/profile, with the middlewares from the pprof package.
	server := http.NewServeMux()
	http.ListenAndServe(":8080", PprofMiddleware(server, "/internal"))
}

func TestOptionsAllowHeader(t *testing.T) {
	h := new(RegexpHandler)
	route := BuildRoute(`^/v1$`)
	h.HandleFunc(route, []string{"GET"}, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Server error"))
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/v1", nil)
	h.ServeHTTP(w, req)
	test.AssertEquals(t, w.Code, http.StatusOK)
	header := w.Header()
	test.AssertEquals(t, header.Get("Allow"), "GET, OPTIONS")
}
