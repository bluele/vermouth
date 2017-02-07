# Vermouth

Vermouth is context friendly web framework. This has dead simple and flexible interface. This project is heavily inspired by [negroni](https://github.com/codegangsta/negroni) and [kami](https://github.com/guregu/kami).

## Features

* flexible middleware stack like negroni

* include httprouter based router

* context-based graceful shutdown support

## Examples

```go
package main

import (
	"github.com/bluele/vermouth"

    "context"
	"fmt"
	"net/http"
)

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
	vm.Serve(":3000")
}

// $ curl 127.0.0.1:3000/
// start:value:end
```

## Middleware stack

vermouth implements hierarchical structures like negroni.
Here is example:

```go
vm.Use("/", Middleware1)
vm.Use("/", Middleware2)
vm.Get("/", HTTPHandler)
```

In above case, normally call stack is:

```
Request ->
Middleware1 -> Middleware2 ->
HTTPHandler ->
Middleware2 -> Middleware1 ->
Response
```

## Graceful shutdown support

Vermouth includes context-based graceful shutdown support.

Vermouth server observe the status of context object. If `context.Done()` returns a value, vermouth will be shutdown gracefully.

See following example:

```go
func main() {
    // This server will be shutdown after 10 sec.
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	vm := vermouth.New().WithContext(ctx)

	vm.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello")
	})
	vm.Serve(":3000")
	log.Println("shutdown...")
}
```

See detail configurations: [godoc](https://godoc.org/github.com/bluele/vermouth#Options),
 [tylerb/graceful](https://github.com/tylerb/graceful)

## Contribution

1. Fork ([https://github.com/bluele/vermouth/fork](https://github.com/bluele/vermouth/fork))
1. Create a feature branch
1. Commit your changes
1. Rebase your local changes against the master branch
1. Run test suite with the `go test ./...` command and confirm that it passes
1. Run `gofmt -s`
1. Create new Pull Request

## Author

**Jun Kimura**

* <http://github.com/bluele>
* <junkxdev@gmail.com>
