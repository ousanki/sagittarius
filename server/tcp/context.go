package tcp

import (
	"context"
	"encoding/json"
	"errors"
	"net"
)

type core func(*Context)

type Context struct {
	ctx       context.Context
	conn      net.Conn
	cores     []core
	header    *Header
	body      *Body
	index     int8
	withTrace int8
}

func newContext() *Context {
	c := &Context{
		withTrace: UnUseTracer,
		index:     0,
		header:    nil,
		body:      nil,
		cores:     nil,
		conn:      nil,
		ctx:       context.TODO(),
	}
	return c
}

func (c *Context) reset() {
	c.withTrace = UnUseTracer
	c.body = nil
	c.header = nil
	c.index = 0
	c.conn = nil
	c.cores = nil
	c.ctx = context.TODO()
}

func (c *Context) Build(ctx context.Context, conn net.Conn) {
	c.ctx = ctx
	c.conn = conn
}

func (c *Context) do() {
	for c.index < int8(len(c.cores)) {
		c.cores[c.index](c)
		c.index++
	}
}

func (c *Context) Ctx() context.Context {
	return c.ctx
}

func (c *Context) Next() {
	c.index++
	for c.index < int8(len(c.cores)) {
		c.cores[c.index](c)
		c.index++
	}
}

func (c *Context) GetHeaderValue(key string) interface{} {
	if v, ok := c.header.values[key]; ok {
		return v
	}
	return nil
}

func (c *Context) SetHeaderValue(key string, value interface{}) {
	if c.header.values == nil {
		c.header.values = make(map[string]interface{})
	}
	c.header.values[key] = value
}

func (c *Context) ReadJSON(data interface{}) error {
	if c.body.buf == nil || len(c.body.buf) == 0 {
		return errors.New("nil buff")
	}
	err := json.Unmarshal(c.body.buf, data)
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) WithTrace(with int8) {
	c.withTrace = with
}

func (c *Context) Write(id int64, data interface{}) error {
	return Write(c.ctx, c.withTrace, c.header.values, id, data, c.conn)
}
