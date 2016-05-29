# Vermouth

Vermouth is [net/context](https://godoc.org/golang.org/x/net/context) friendly web framework. This has dead simple and flexible interface. This project is heavily inspired by [negroni](https://github.com/codegangsta/negroni) and [kami](https://github.com/guregu/kami).

## Features

* support net/context based handler

* flexible middleware stack like negroni

* include httprouter based router

## Why?

I had searched net/context based web framework. And I found kami,
but it cannot support many existing net/http middleware that made by people use net/http and negroni.
For that reason, I decide to develop a new net/context based web framework that resolve the issue.

## Examples

```go
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
