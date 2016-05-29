package vermouth

import (
	"net"

	"github.com/tylerb/graceful"
)

// Serve is a convenience function that runs the vermouth stack as an HTTP
// server. The addr string takes the same format as http.ListenAndServe.
func (vm *Vermouth) Serve(addr string) error {
	srv := vm.newServer()
	srv.Addr = addr
	go vm.observeContext(srv)
	return srv.ListenAndServe()
}

// ServeListener is like Serve, but runs vermouth on top of an arbitrary net.Listener.
func (vm *Vermouth) ServeListener(l net.Listener) error {
	srv := vm.newServer()
	go vm.observeContext(srv)
	return srv.Serve(l)
}

// observeContext observe the status for top level context object.
// If context is done, shutdown a server gracefully.
func (vm *Vermouth) observeContext(srv *graceful.Server) {
	select {
	case <-vm.ctx.Done():
		srv.Stop(srv.Timeout)
	case <-srv.StopChan():
	}
}
