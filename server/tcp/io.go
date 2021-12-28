package tcp

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/opentracing/opentracing-go"
	"io"
	"net"
)

const (
	UseTracer   = 1
	UnUseTracer = 0
)

type headerBase struct {
	WithTrace int8
	Len       int64
	ID        int64
}

func (hb *headerBase) GetID() int64 {
	return hb.ID
}

func (hb *headerBase) IsWithTrace() bool {
	return hb.WithTrace == UseTracer
}

type Header struct {
	headerBase
	buf    []byte
	values map[string]interface{}
}

type bodyBase struct {
	Len int64
}

type Body struct {
	bodyBase
	buf []byte
}

func Read(ctx context.Context, conn *conn) (*Context, error) {
	c := conn.server.pool.Get().(*Context)
	c.Build(ctx, conn.c)
	// read header
	h, err := readHeader(c.conn)
	if err != nil {
		return nil, err
	}
	c.header = h
	// tracer
	if h.IsWithTrace() {
		spCtx, err := opentracing.GlobalTracer().Extract(
			opentracing.Binary,
			c.conn,
		)
		if err != nil {
			return nil, err
		} else {
			// reset ctx
			span := opentracing.GlobalTracer().StartSpan(fmt.Sprintf("%d", h.GetID()), opentracing.ChildOf(spCtx))
			c.ctx = opentracing.ContextWithSpan(c.ctx, span)
		}
	}
	// read body
	b, err := readBody(c.conn)
	if err != nil {
		return nil, err
	}
	c.body = b
	return c, nil
}

func readHeader(r io.Reader) (*Header, error) {
	// get header len
	h := new(Header)
	err := binary.Read(r, binary.BigEndian, &h.headerBase)
	if err != nil {
		return nil, err
	}
	// get header buff
	h.buf = make([]byte, h.Len)
	_, err = io.ReadFull(r, h.buf)
	if err != nil {
		return nil, err
	}
	// get header values
	if h.values == nil {
		h.values = make(map[string]interface{})
	}
	err = json.Unmarshal(h.buf, &h.values)
	if err != nil {
		return nil, err
	}
	return h, nil
}

func readBody(r io.Reader) (*Body, error) {
	// get header len
	b := Body{}
	err := binary.Read(r, binary.BigEndian, &b.bodyBase)
	if err != nil {
		return nil, err
	}
	// get header buff
	b.buf = make([]byte, b.Len)
	_, err = io.ReadFull(r, b.buf)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func Write(
	ctx context.Context,
	withTrace int8,
	headerValues map[string]interface{},
	id int64,
	data interface{},
	conn net.Conn) error {
	hb := headerBase{
		ID:        id,
		WithTrace: withTrace,
	}
	hv, err := json.Marshal(headerValues)
	if err != nil {
		return err
	}
	hb.Len = int64(len(hv))
	// write header
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, hb)
	if err != nil {
		return err
	}
	err = binary.Write(buf, binary.BigEndian, hv)
	if err != nil {
		return err
	}
	// write span
	if hb.IsWithTrace() {
		span := opentracing.SpanFromContext(ctx)
		if span == nil {
			span = opentracing.StartSpan(fmt.Sprintf("%d", hb.GetID()))
		}
		err = opentracing.GlobalTracer().Inject(
			span.Context(),
			opentracing.Binary,
			buf,
		)
		if err != nil {
			return err
		}
	}
	// write body
	bb := bodyBase{}
	bv, err := json.Marshal(data)
	if err != nil {
		return err
	}
	bb.Len = int64(len(bv))
	err = binary.Write(buf, binary.BigEndian, bb)
	if err != nil {
		return err
	}
	err = binary.Write(buf, binary.BigEndian, bv)
	if err != nil {
		return err
	}
	message := buf.Bytes()
	var offset int
	for {
		n, err := conn.Write(message[offset:])
		if err != nil {
			return err
		}
		if offset+n >= len(message) {
			break
		}
		offset += n
	}
	return nil
}
