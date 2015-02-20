package client

import (
	"net"
	"errors"
	"log"
	"bufio"
)

const (
	Version5    = byte(5)
	MethodBare  = byte(0)
	MethodUnPwd = byte(2)
	MethodUnPwdVer = byte(1)
	MethodUnPwdStatusOk = byte(0)
)

var (
	ErrResolveAddr    = errors.New("Client: Failed to resolve local address")
	ErrResolveSrvAddr = errors.New("Client: Failed to resoleve server address")
	ErrCannotListen   = errors.New("Client: Failed to create listener")
)

type Client struct {
	Addr *net.TCPAddr
	SrvAddr *net.TCPAddr
	Conn *net.TCPConn
	Un  []byte
	Pwd []byte
}

func (c *Client) SetAddr(addr string) error {
	if ad, err := net.ResolveTCPAddr("tcp", addr); err != nil {
		return ErrResolveAddr
	}else {
		c.Addr = ad
	}

	return nil
}

func (c *Client) SetSrvAddr(addr string) error {
	if ad, err := net.ResolveTCPAddr("tcp", addr); err != nil {
		return ErrResolveSrvAddr
	}else {
		c.SrvAddr = ad
		return nil
	}
}

func (c *Client) ListenAndServe() error {
	var (
		l net.Listener
		err error
	)

	if l, err = net.ListenTCP("tcp", c.Addr); err != nil {
		return ErrCannotListen
	}

	return c.Serve(l)
}

func (c *Client) Serve(l net.Listener) error {
	defer l.Close()

	var (
		nec net.Conn
		err error
		ne   net.Error
		ok bool
		conn *Conn
	)

	for {
		nec, err = l.Accept()
		if err != nil {
			if ne, ok = err.(net.Error); ok && ne.Temporary() {
				continue
			}

			return err
		}

		if conn, err = c.newConn(nec); err != nil {
			continue
		}

		go conn.Serve()
	}

	return nil
}

func (c *Client) newConn(nec net.Conn) (*Conn, error) {
	conn := new(Conn)

	conn.Client = c
	conn.Conn = nec
	conn.ConnBr = bufio.NewReader(nec)

	return conn, nil
}

func (c *Client) Log(args ...interface{}) {
	log.Println(args...)
}
