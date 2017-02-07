package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/bluele/vermouth"
)

const addr = ":3000"

func main() {
	vm := vermouth.New()
	vm.Use("/", func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		fmt.Fprint(w, "start:")
		defer fmt.Fprint(w, ":end")
		// call next middleware
		next(w, vermouth.WithValue(r, "key", "value"))
	})
	vm.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, r.Context().Value("key").(string))
	})

	log.Printf("serving at %v", addr)
	vm.Serve(addr)
}
