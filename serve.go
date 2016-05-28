package vermouth

import (
	"log"
	"net"
	"os"

	"github.com/tylerb/graceful"
)

// Run is a convenience function that runs the vermouth stack as an HTTP
// server. The addr string takes the same format as http.ListenAndServe.
func (vm *Vermouth) Run(addr string) {
	l := log.New(os.Stdout, "[vermouth] ", 0)
	l.Printf("listening on %s", addr)
	l.Fatal(vm.Serve(addr))
}

func (vm *Vermouth) Serve(addr string) error {
	srv := vm.newServer()
	srv.Addr = addr
	go vm.observeContext(srv)
	return srv.ListenAndServe()
}

func (vm *Vermouth) ServeListener(l net.Listener) error {
	srv := vm.newServer()
	go vm.observeContext(srv)
	return srv.Serve(l)
}

func (vm *Vermouth) observeContext(srv *graceful.Server) {
	select {
	case <-vm.ctx.Done():
		srv.Stop(srv.Timeout)
	}
}
