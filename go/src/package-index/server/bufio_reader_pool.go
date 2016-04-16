package server

import (
	"bufio"
	"io"
	"sync"
)

// bufioReaderPool is a convenience wrapper around a sync.Pool of
// bufio.Readers of size BufSize.
type bufioReaderPool struct {
	BufSize int
	pool    sync.Pool
}

// Get draws from the pool, or if there is no buffer available makes a new one
// with size p.BufSize.
func (p *bufioReaderPool) Get(r io.Reader) *bufio.Reader {
	if v := p.pool.Get(); v != nil {
		buf := v.(*bufio.Reader)
		buf.Reset(r)
		return buf
	}
	return bufio.NewReaderSize(r, p.BufSize)
}

// Put returns a buffer from Get to the pool so it may be garbage collected.
func (p *bufioReaderPool) Put(buf *bufio.Reader) {
	buf.Reset(nil)
	p.pool.Put(buf)
}
