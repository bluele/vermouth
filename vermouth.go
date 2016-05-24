package vermouth

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/context"
)

// Handler is an interface that objects can implement to be registered to serve as middleware
// in the Vermouth middleware stack.
// ServeHTTP should yield to the next middleware in the chain by invoking the next vermouth.ContextHandlerFunc
// passed in.
//
// If the Handler writes to the ResponseWriter, the next vermouth.ContextHandlerFunc should not be invoked.
type Handler interface {
	ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request, next ContextHandlerFunc)
}

// ContextHandler is like http.Handler but supports context.
type ContextHandler interface {
	ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request)
}

type (
	// HandlerType is the type of Handlers and types that vermouth internally converts to
	// ContextHandlerFunc. In order to provide an expressive API, this type is an alias for
	// interface{} that is named for the purposes of documentation, however only the
	// following concrete types are accepted:
	// 	- func(context.Context, http.ResponseWriter, *http.Request)
	// 	- func(http.ResponseWriter, *http.Request)
	// 	- types that implement ContextHandler
	// 	- types that implement http.Handler
	HandlerType interface{}

	// MiddlewareType represents types that vermouth can convert to Middleware.
	// vermouth will try its best to convert standard, non-context middleware.
	// See the Use function for important information about how vermouth middleware is run.
	// The following concrete types are accepted:
	// 	- types that implement Handler
	// 	- func(context.Context, http.ResponseWriter, *http.Request, ContextHandlerFunc)
	// 	- func(http.ResponseWriter, *http.Request, http.HandlerFunc)
	MiddlewareType interface{}

	// ContextHandlerFunc is like http.HandlerFunc with context.
	ContextHandlerFunc func(context.Context, http.ResponseWriter, *http.Request)
)

// HandlerFunc is an adapter to allow the use of ordinary functions as Vermouth handlers.
// If f is a function with the appropriate signature, HandlerFunc(f) is a Handler object that calls f.
type HandlerFunc func(ctx context.Context, w http.ResponseWriter, r *http.Request, next ContextHandlerFunc)

func (h HandlerFunc) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {
	h(ctx, w, r, next)
}

type middleware struct {
	handler Handler
	next    *middleware
}

func (m middleware) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	m.handler.ServeHTTP(ctx, w, r, m.next.ServeHTTP)
}

// Vermouth object
type Vermouth struct {
	ctx      context.Context
	router   *Router
	handlers []Handler
}

// New creates a new independent router and middleware stack.
func New() *Vermouth {
	return &Vermouth{
		ctx:    context.Background(),
		router: NewRouter(),
	}
}

// SetContext sets a root net/context object.
// All request context will be derive from this context.
func (vm *Vermouth) SetContext(ctx context.Context) *Vermouth {
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
	md.ServeHTTP(vm.ctx, NewResponseWriter(w), r)
}

func (vm *Vermouth) HandlerFunc() http.HandlerFunc {
	md := build(append(vm.handlers, wrapHandler(vm.router)))
	return func(w http.ResponseWriter, r *http.Request) {
		md.ServeHTTP(vm.ctx, w, r)
	}
}

// Middlewares returns registered handlers.
func (vm *Vermouth) Middlewares() []Handler {
	return vm.handlers
}

// Run is a convenience function that runs the vermouth stack as an HTTP
// server. The addr string takes the same format as http.ListenAndServe.
func (vm *Vermouth) Run(addr string) {
	l := log.New(os.Stdout, "[vermouth] ", 0)
	l.Printf("listening on %s", addr)
	l.Fatal(http.ListenAndServe(addr, vm))
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

func wrapHandler(handler ContextHandler) Handler {
	return HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {
		handler.ServeHTTP(ctx, w, r)
		next(ctx, w, r)
	})
}

func makeRoutingHandler(pattern string, handler Handler) Handler {
	isWildcard := pattern == "" || pattern == "/"
	if !isWildcard && pattern[len(pattern)-1] != '/' {
		pattern += "/"
	}
	return HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {
		if r == nil || isWildcard {
			handler.ServeHTTP(ctx, w, r, next)
		} else {
			if strings.HasPrefix(r.URL.Path+"/", pattern) {
				handler.ServeHTTP(ctx, w, r, next)
			} else {
				next(ctx, w, r)
			}
		}
	})
}

func wrapHandlerFunc(handler HandlerType) ContextHandlerFunc {
	switch h := handler.(type) {
	case func(context.Context, http.ResponseWriter, *http.Request):
		return h
	case func(http.ResponseWriter, *http.Request):
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			h(w, r)
		}
	case ContextHandler:
		return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			h.ServeHTTP(ctx, w, r)
		}
	case http.Handler:
		return func(_ context.Context, w http.ResponseWriter, r *http.Request) {
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
	case func(context.Context, http.ResponseWriter, *http.Request, ContextHandlerFunc):
		return HandlerFunc(m)
	case func(http.ResponseWriter, *http.Request, http.HandlerFunc):
		return HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {
			m(w, r, func(w http.ResponseWriter, r *http.Request) {
				next(ctx, w, r)
			})
		})
	default:
		panic(fmt.Sprintf("Unknown middleware type: %T", m))
	}
}

func voidMiddleware() middleware {
	return middleware{
		HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {}),
		&middleware{},
	}
}
