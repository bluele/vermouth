package main

import (
	"github.com/bluele/vermouth"
	"golang.org/x/net/context"

	"fmt"
	"net/http"
)

func main() {
	vm := vermouth.New()
	vm.Use("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request, next vermouth.ContextHandlerFunc) {
		fmt.Fprint(w, "start:")
		defer fmt.Fprint(w, ":end")
		// call next middleware
		next(context.WithValue(ctx, "key", "value"), w, r)
	})
	vm.Get("/", func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, ctx.Value("key").(string))
	})
	vm.Serve(":3000")
}
