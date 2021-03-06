package vermouth

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/tylerb/graceful"
)

// Handler is an interface that objects can implement to be registered to serve as middleware
// in the Vermouth middleware stack.
// ServeHTTP should yield to the next middleware in the chain by invoking the next http.HandlerFunc
// passed in.
//
// If the Handler writes to the ResponseWriter, the next http.HandlerFunc should not be invoked.
type Handler interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)
}

type (
	// HandlerType is the type of Handlers and types that vermouth internally converts to
	// http.HandlerFunc. In order to provide an expressive API, this type is an alias for
	// interface{} that is named for the purposes of documentation, however only the
	// following concrete types are accepted:
	// 	- func(http.ResponseWriter, *http.Request)
	// 	- types that implement http.Handler
	HandlerType interface{}

	// MiddlewareType represents types that vermouth can convert to Middleware.
	// vermouth will try its best to convert standard, non-context middleware.
	// See the Use function for important information about how vermouth middleware is run.
	// The following concrete types are accepted:
	// 	- types that implement Handler
	// 	- func(http.ResponseWriter, *http.Request, http.HandlerFunc)
	MiddlewareType interface{}
)

// HandlerFunc is an adapter to allow the use of ordinary functions as Vermouth handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a Handler object that calls f.
type HandlerFunc func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)

func (h HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	h(w, r, next)
}

type middleware struct {
	handler Handler
	next    *middleware
}

func (m middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.handler.ServeHTTP(w, r, m.next.ServeHTTP)
}

// Vermouth object
type Vermouth struct {
	ctx      context.Context
	router   *Router
	handlers []Handler
	Options  *Options
}

// Options for vermouth app
// This is used by initializing http.Server and graceful.Server
type Options struct {
	ReadTimeout    time.Duration // maximum duration before timing out read of the request
	WriteTimeout   time.Duration // maximum duration before timing out write of the response
	MaxHeaderBytes int           // maximum size of request headers, DefaultMaxHeaderBytes if 0
	TLSConfig      *tls.Config   // optional TLS config, used by ListenAndServeTLS

	// TLSNextProto optionally specifies a function to take over
	// ownership of the provided TLS connection when an NPN
	// protocol upgrade has occurred.  The map key is the protocol
	// name negotiated. The Handler argument should be used to
	// handle HTTP requests and will initialize the Request's TLS
	// and RemoteAddr if not already set.  The connection is
	// automatically closed when the function returns.
	// If TLSNextProto is nil, HTTP/2 support is enabled automatically.
	TLSNextProto map[string]func(*http.Server, *tls.Conn, http.Handler)

	// ConnState specifies an optional callback function that is
	// called when a client connection changes state. See the
	// ConnState type and associated constants for details.
	ConnState func(net.Conn, http.ConnState)

	// Timeout is the duration to allow outstanding requests to survive
	// before forcefully terminating them.
	GracefulTimeout time.Duration

	// Limit the number of outstanding requests
	ListenLimit int

	// ShutdownInitiated is an optional callback function that is called
	// when shutdown is initiated. It can be used to notify the client
	// side of long lived connections (e.g. websockets) to reconnect.
	ShutdownInitiated func()

	// NoSignalHandling prevents graceful from automatically shutting down
	// on SIGINT and SIGTERM. If set to true, you must shut down the server
	// manually with Stop().
	NoSignalHandling bool

	// ErrorLog specifies an optional logger for errors accepting
	// connections and unexpected behavior from handlers.
	// If nil, logging goes to os.Stderr via the log package's
	// standard logger.
	ErrorLog *log.Logger
}

// New creates a new independent router and middleware stack.
func New() *Vermouth {
	return &Vermouth{
		ctx:    context.Background(),
		router: NewRouter(),

		Options: defaultOptions(),
	}
}

// WithContext sets a root context object.
// All request context will be derive from this context.
func (vm *Vermouth) WithContext(ctx context.Context) *Vermouth {
	vm.ctx = ctx
	return vm
}

// SetRouter sets a router object
func (vm *Vermouth) SetRouter(router *Router) *Vermouth {
	vm.router = router
	return vm
}

// Use adds a Handler onto the middleware stack.
// Handlers are invoked in the order they are added to a Vermouth.
func (vm *Vermouth) Use(pattern string, mw MiddlewareType) *Vermouth {
	handler := wrapMiddlewareFunc(mw)
	vm.handlers = append(vm.handlers, makeRoutingHandler(pattern, handler))
	return vm
}

// Get registers a GET handler under the given path.
func (vm *Vermouth) Get(pattern string, handler HandlerType) *Vermouth {
	return vm.Handle("GET", pattern, handler)
}

// Post registers a GET handler under the given path.
func (vm *Vermouth) Post(pattern string, handler HandlerType) *Vermouth {
	return vm.Handle("POST", pattern, handler)
}

// Handle registers an arbitrary method handler under the given path.
func (vm *Vermouth) Handle(method, pattern string, handler HandlerType) *Vermouth {
	vm.router.Handle(method, pattern, wrapHandlerFunc(handler))
	return vm
}

func (vm *Vermouth) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	md := build(append(vm.handlers, wrapHandler(vm.router)))
	md.ServeHTTP(NewResponseWriter(w), r.WithContext(vm.ctx))
}

func (vm *Vermouth) HandlerFunc() http.HandlerFunc {
	md := build(append(vm.handlers, wrapHandler(vm.router)))
	return func(w http.ResponseWriter, r *http.Request) {
		md.ServeHTTP(w, r.WithContext(vm.ctx))
	}
}

// Middlewares returns registered handlers.
func (vm *Vermouth) Middlewares() []Handler {
	return vm.handlers
}

// NewServer returns a new server implements http.Server.
func (vm *Vermouth) NewServer() *graceful.Server {
	opts := vm.Options
	if opts == nil {
		opts = defaultOptions()
	}
	srv := &http.Server{
		Handler:        vm,
		ReadTimeout:    opts.ReadTimeout,
		WriteTimeout:   opts.WriteTimeout,
		MaxHeaderBytes: opts.MaxHeaderBytes,
		TLSConfig:      opts.TLSConfig,
		TLSNextProto:   opts.TLSNextProto,
		ErrorLog:       opts.ErrorLog,
	}
	return &graceful.Server{
		Server:           srv,
		Timeout:          opts.GracefulTimeout,
		ListenLimit:      opts.ListenLimit,
		ConnState:        opts.ConnState,
		NoSignalHandling: opts.NoSignalHandling,
	}
}

// defaultOptions returns default options.
func defaultOptions() *Options {
	return &Options{
		NoSignalHandling: true,
	}
}

func build(handlers []Handler) middleware {
	var next middleware

	if len(handlers) == 0 {
		return voidMiddleware()
	} else if len(handlers) > 1 {
		next = build(handlers[1:])
	} else {
		next = voidMiddleware()
	}

	return middleware{handlers[0], &next}
}

func wrapHandler(handler http.Handler) Handler {
	return HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		handler.ServeHTTP(w, r)
		next(w, r)
	})
}

func makeRoutingHandler(pattern string, handler Handler) Handler {
	isWildcard := pattern == "" || pattern == "/"
	if !isWildcard && pattern[len(pattern)-1] != '/' {
		pattern += "/"
	}
	return HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if r == nil || isWildcard {
			handler.ServeHTTP(w, r, next)
		} else {
			if strings.HasPrefix(r.URL.Path+"/", pattern) {
				handler.ServeHTTP(w, r, next)
			} else {
				next(w, r)
			}
		}
	})
}

func wrapHandlerFunc(handler HandlerType) http.HandlerFunc {
	switch h := handler.(type) {
	case func(http.ResponseWriter, *http.Request):
		return h
	case http.Handler:
		return func(w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(w, r)
		}
	default:
		panic(fmt.Sprintf("Unknown handler type: %T", h))
	}
}

func wrapMiddlewareFunc(mw MiddlewareType) Handler {
	switch m := mw.(type) {
	case Handler:
		return m
	case func(http.ResponseWriter, *http.Request, http.HandlerFunc):
		return HandlerFunc(m)
	default:
		panic(fmt.Sprintf("Unknown middleware type: %T", m))
	}
}

func voidMiddleware() middleware {
	return middleware{
		HandlerFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {}),
		&middleware{},
	}
}

func WithValue(r *http.Request, key, value interface{}) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), key, value))
}
