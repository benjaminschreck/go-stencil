package stencil

import (
	"compress/flate"
	"io"
	"sync"
)

var bytesBufferPool = sync.Pool{
	New: func() interface{} {
		return new(bufferWithBytes)
	},
}

var flateWriterPool sync.Pool

type bufferWithBytes struct {
	buf []byte
}

func getBufferBytes() []byte {
	pooled := bytesBufferPool.Get().(*bufferWithBytes)
	if pooled.buf == nil {
		pooled.buf = make([]byte, 0, 32*1024)
	}
	return pooled.buf[:0]
}

func putBufferBytes(buf []byte) {
	bytesBufferPool.Put(&bufferWithBytes{buf: buf[:0]})
}

type pooledFlateWriteCloser struct {
	writer *flate.Writer
}

func newPooledFlateWriter(out io.Writer) (io.WriteCloser, error) {
	if pooled := flateWriterPool.Get(); pooled != nil {
		writer := pooled.(*flate.Writer)
		writer.Reset(out)
		return &pooledFlateWriteCloser{writer: writer}, nil
	}

	writer, err := flate.NewWriter(out, flate.BestSpeed)
	if err != nil {
		return nil, err
	}
	return &pooledFlateWriteCloser{writer: writer}, nil
}

func (p *pooledFlateWriteCloser) Write(data []byte) (int, error) {
	return p.writer.Write(data)
}

func (p *pooledFlateWriteCloser) Close() error {
	err := p.writer.Close()
	p.writer.Reset(io.Discard)
	flateWriterPool.Put(p.writer)
	p.writer = nil
	return err
}
