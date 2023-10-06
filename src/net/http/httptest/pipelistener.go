// Copyright 2023 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httptest

import (
	"context"
	"errors"
	"net"
	"sync"
)

// A pipeListener is a net.Listener implementation that serves
// in-memory pipes.
type pipeListener struct {
	addr pipeAddr

	close sync.Once
	done  chan struct{}
	ch    chan net.Conn
}

func newPipeListener(addr string) *pipeListener {
	return &pipeListener{
		addr: pipeAddr(addr),
		done: make(chan struct{}),
		c:    make(chan net.Conn),
	}
}

func (lis *pipeListener) Accept() (net.Conn, error) {
	aerr := func(err error) error {
		return &net.OpError{
			Op:   "accept",
			Net:  lis.addr.Network(),
			Addr: lis.addr,
			Err:  err,
		}
	}
	select {
	case <-lis.done:
		return nil, aerr(errors.New("closed"))
	case s := <-lis.ch:
		return s, nil
	}
}

func (lis *pipeListener) Close() (err error) {
	err = errors.New("closed")
	lis.close.Do(func() {
		close(lis.done)
		err = nil
	})
	return
}

func (lis *pipeListener) Addr() net.Addr {
	return pipeAddr(lis.addr)
}

// Dial "dials" the listener, creating a pipe. It returns the client's
// end of the connection and causes a waiting Accept call to return
// the server's end.
func (lis *pipeListener) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	derr := func(err error) error {
		return &net.OpError{
			Op:  "dial",
			Net: lis.addr.Network(),
			Err: err,
		}
	}
	s, c := net.Pipe()
	select {
	case <-ctx.Done():
		return nil, derr(ctx.Err())
	case lis.ch <- s:
		return c, nil
	case <-lis.done:
		return nil, derr(errors.New("closed"))
	}
	return c, nil
}

type pipeAddr string

func (pipeAddr) Network() string {
	return "pipe"
}

func (addr pipeAddr) String() string {
	return string(addr)
}
