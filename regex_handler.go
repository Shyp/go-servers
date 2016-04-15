// A simple http.Handler that can match wildcard routes, and call the
// appropriate handler.
package server

import (
	"bytes"
	"encoding/json"
	"expvar"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/http/pprof"
	"os"
	"regexp"
	"strings"
	"sync"
)

type route struct {
	pattern *regexp.Regexp
	methods []string
	handler http.Handler
}

func BuildRoute(regex string) *regexp.Regexp {
	route, err := regexp.Compile(regex)
	if err != nil {
		log.Fatal(err)
	}
	return route
}

// RegexpHandler is a HTTP handler that can handle regex routes. If a route
// doesn't match, a 404 error message is returned.
type RegexpHandler struct {
	routes []*route
}

func (h *RegexpHandler) Handler(pattern *regexp.Regexp, methods []string, handler http.Handler) {
	h.routes = append(h.routes, &route{
		pattern: pattern,
		methods: methods,
		handler: handler,
	})
}

// JSONMiddleware is a middleware that adds the application/json content type to
// a response.
func JSONMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		h.ServeHTTP(w, r)
	})
}

var mu sync.Mutex

// DebugRequestBodyHandler prints all incoming and outgoing HTTP traffic if the
// DEBUG_HTTP_TRAFFIC environment variable is set to true.
func DebugRequestBodyMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("DEBUG_HTTP_TRAFFIC") == "true" {
			mu.Lock()
			defer mu.Unlock()
			// You want to write the entire thing in one Write.
			b := new(bytes.Buffer)
			bits, err := httputil.DumpRequest(r, true)
			if err != nil {
				_, _ = b.WriteString(err.Error())
			} else {
				_, _ = b.Write(bits)
			}
			res := httptest.NewRecorder()
			h.ServeHTTP(res, r)

			_, _ = b.WriteString(fmt.Sprintf("HTTP/1.1 %d\r\n", res.Code))
			_ = res.HeaderMap.Write(b)
			w.WriteHeader(res.Code)
			for k, v := range res.HeaderMap {
				w.Header()[k] = v
			}
			_, _ = b.WriteString("\r\n")
			writer := io.MultiWriter(w, b)
			_, _ = res.Body.WriteTo(writer)
			_, _ = b.WriteTo(os.Stderr)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}

// ExpvarMiddleware exports an expvar route at the given endpoint. If the
// endpoint is the empty string, the endpoint will be exposed at /debug/vars.
func ExpvarMiddleware(h http.Handler, endpoint string) http.Handler {
	if endpoint == "" {
		endpoint = "/debug/vars"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != endpoint {
			h.ServeHTTP(w, r)
		} else {
			// Implementation here is taken from the expvar package.
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			fmt.Fprintf(w, "{\n")
			first := true
			expvar.Do(func(kv expvar.KeyValue) {
				if !first {
					fmt.Fprintf(w, ",\n")
				}
				first = false
				fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
			})
			fmt.Fprintf(w, "\n}\n")
		}
	})
}

// PprofMiddleware exposes every endpoint from net/http/pprof, optionally
// with the given prefix. If the prefix is the empty string, default to
// /debug/pprof.
func PprofMiddleware(h http.Handler, prefix string) http.Handler {
	if prefix == "" {
		prefix = "/debug/pprof"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, prefix) {
			h.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == fmt.Sprintf("%s/cmdline", prefix) {
			pprof.Cmdline(w, r)
		} else if r.URL.Path == fmt.Sprintf("%s/profile", prefix) {
			pprof.Profile(w, r)
		} else if r.URL.Path == fmt.Sprintf("%s/symbol", prefix) {
			pprof.Symbol(w, r)
		} else if r.URL.Path == fmt.Sprintf("%s/trace", prefix) {
			pprof.Trace(w, r)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}

func (h *RegexpHandler) HandleFunc(pattern *regexp.Regexp, methods []string, handler func(http.ResponseWriter, *http.Request)) {
	h.routes = append(h.routes, &route{
		pattern: pattern,
		methods: methods,
		handler: http.HandlerFunc(handler),
	})
}

func (h *RegexpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range h.routes {
		if route.pattern.MatchString(r.URL.Path) {
			upperMethod := strings.ToUpper(r.Method)
			for _, method := range route.methods {
				if strings.ToUpper(method) == upperMethod {
					route.handler.ServeHTTP(w, r)
					return
				}
			}
			if upperMethod == "OPTIONS" {
				methods := strings.Join(append(route.methods, "OPTIONS"), ", ")
				w.Header().Set("Allow", methods)
				return
			} else {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusMethodNotAllowed)
				json.NewEncoder(w).Encode(new405(r))
			}
			return
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(new404(r))
}
