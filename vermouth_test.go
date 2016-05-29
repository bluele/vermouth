package vermouth

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"golang.org/x/net/context"
)

/* Test Helpers */
func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func refute(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		t.Errorf("Did not expect %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func TestVermouthRun(t *testing.T) {
	// just test that Run doesn't bomb
	go New().Serve(":3000")
}

func TestVermouthServeHTTP(t *testing.T) {
	result := ""
	response := httptest.NewRecorder()

	n := New()
	n.Use("", HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {
		result += "foo"
		next(ctx, w, r)
		result += "ban"
	}))
	n.Use("", HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {
		result += "bar"
		next(ctx, w, r)
		result += "baz"
	}))
	n.Use("", HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {
		result += "bat"
		w.WriteHeader(http.StatusBadRequest)
	}))

	n.ServeHTTP(response, (*http.Request)(nil))

	expect(t, result, "foobarbatbazban")
	expect(t, response.Code, http.StatusBadRequest)
}

// Ensures that a Vermouth middleware chain
// can correctly return all of its handlers.
func TestMiddlewares(t *testing.T) {
	response := httptest.NewRecorder()
	n := New()
	handlers := n.Middlewares()
	expect(t, 0, len(handlers))

	n.Use("", HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request, next ContextHandlerFunc) {
		w.WriteHeader(http.StatusOK)
	}))

	// Expects the length of handlers to be exactly 1
	// after adding exactly one handler to the middleware chain
	handlers = n.Middlewares()
	expect(t, 1, len(handlers))

	// Ensures that the first handler that is in sequence behaves
	// exactly the same as the one that was registered earlier
	handlers[0].ServeHTTP(context.Background(), response, (*http.Request)(nil), nil)
	expect(t, response.Code, http.StatusOK)
}
