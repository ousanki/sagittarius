package tcp

import (
	"context"
	"errors"
	"fmt"
	"github.com/ousanki/sagittarius/core/log"
	"io"
	"net"
	"sync"
	"time"
)

var genLogger *log.Logger

func init() {
	genLogger = log.New("gen")
	genLogger.WithOptions(
		log.SetRotation(log.RotationDay),
		log.SetPath("./log"),
		log.SetFormat(log.ConsoleFormat),
	)
}

type conn struct {
	server     *Engine
	c          net.Conn
	ctx        context.Context
	cancel     func()
	remoteAddr string
}

func (c *conn) serve() {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			ctx, err := Read(c.ctx, c)
			if err != nil {
				if err == io.ErrUnexpectedEOF || err == io.EOF {
					return
				}
				genLogger.Write(ctx.Ctx(), "tcp conn read error, remote:%s, err:%v", c.remoteAddr, err)
			} else {
				ctx.cores = c.server.findCore(ctx.header.GetID())
				ctx.do()
			}
		}
	}
}

type Engine struct {
	*Group
	Addr  string
	Proto string

	mu         sync.Mutex
	listener   net.Listener
	activeConn map[*conn]struct{}
	doneChan   chan struct{}
	handlers   map[int64][]core
	pool       sync.Pool
}

func NewApp(proto string) *Engine {
	engine := &Engine{
		Proto: proto,
	}
	group := &Group{
		svr:  engine,
		root: true,
	}
	engine.Group = group
	engine.pool.New = func() interface{} {
		return newContext()
	}
	return engine
}

func (s *Engine) Run(port string) error {
	s.Addr = port
	port = fmt.Sprintf("0.0.0.0:%s", s.Addr)
	addr, err := net.ResolveTCPAddr(s.Proto, port)
	if err != nil {
		return err
	}

	listener, err := net.ListenTCP(s.Proto, addr)
	if err != nil {
		return err
	}
	s.listener = listener

	return s.Serve(listener)
}

func (s *Engine) Serve(l net.Listener) error {
	baseCtx := context.Background()
	for {
		c, err := l.Accept()
		if err != nil {
			select {
			case <-s.getDoneChan():
				return errors.New("tcp: Server closed")
			}
			continue
		}
		ctx := context.WithValue(baseCtx, "accept", time.Now().Format("2006-01-02 15:04:05.000"))
		ctx = context.WithValue(ctx, "remote", c.RemoteAddr().String())

		ctx, fn := context.WithCancel(ctx)
		cn := conn{
			ctx:        ctx,
			cancel:     fn,
			c:          c,
			server:     s,
			remoteAddr: c.RemoteAddr().String(),
		}
		go func() {
			s.trackConn(&cn, true)
			defer s.trackConn(&cn, false)

			cn.serve()
		}()
	}
}

func (s *Engine) getDoneChan() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.doneChan == nil {
		s.doneChan = make(chan struct{})
	}
	return s.doneChan
}

func (s *Engine) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for conn, _ := range s.activeConn {
		conn.cancel()
		delete(s.activeConn, conn)
	}
	s.doneChan <- struct{}{}
}

func (s *Engine) trackConn(c *conn, add bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeConn == nil {
		s.activeConn = make(map[*conn]struct{})
	}
	if add {
		s.activeConn[c] = struct{}{}
	} else {
		delete(s.activeConn, c)
	}
}

func (s *Engine) addCore(id int64, cores ...core) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.handlers == nil {
		s.handlers = make(map[int64][]core)
	}

	if _, has := s.handlers[id]; has {
		panic(fmt.Sprintf("server router id:%d already exist", id))
	}
	s.handlers[id] = cores
}

func (s *Engine) findCore(id int64) []core {
	if _, has := s.handlers[id]; !has {
		return nil
	}
	return s.handlers[id]
}
