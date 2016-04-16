package server

import (
	"bufio"
	"testing"
)

type nopReader struct{}

func (n nopReader) Read(b []byte) (int, error) {
	return len(b), nil
}

func TestBufioReaderPool(t *testing.T) {
	p := &bufioReaderPool{BufSize: 20}
	buf := p.Get(nopReader{})
	if _, err := buf.Peek(p.BufSize + 2); err != bufio.ErrBufferFull {
		t.Fatal(err)
	}
	if _, err := buf.Peek(p.BufSize); err != nil {
		t.Fatal(err)
	}
	p.Put(buf)
}
