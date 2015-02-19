package server

import (
	"bufio"
	"errors"
	"log"
	"net"
	"time"
)

type Server struct {
	Cfg       *Config
	StartTime time.Time
}

var (
	ErrCannotListen = errors.New("Faild to create TCP listener.")
)

func (s *Server) ListenAndServe() error {
	var (
		l    *net.TCPListener
		addr *net.TCPAddr
		err  error
	)

	addr = &net.TCPAddr{
		[]byte{127, 0, 0, 1},
		int(s.Cfg.ServerPort),
		"",
	}

	if l, err = net.ListenTCP("tcp", addr); err != nil {
		return ErrCannotListen
	}

	return s.Serve(l)
}

func (s *Server) Serve(l net.Listener) error {
	defer l.Close()

	var (
		nec  net.Conn
		err  error
		ne   net.Error
		ok   bool
		conn *conn
	)

	for {
		nec, err = l.Accept()
		if err != nil {
			if ne, ok = err.(net.Error); ok && ne.Temporary() {
				continue
			}

			return err
		}

		if conn, err = s.NewConn(nec); err != nil {
			continue
		}

		go conn.serve()
	}
}

func (s *Server) NewConn(netConn net.Conn) (*conn, error) {
	c := new(conn)

	c.netConn = netConn
	c.server = s
	c.br = bufio.NewReader(netConn)

	return c, nil
}

func (s *Server) Log(args ...interface{}) {
	log.Println(args...)
}
