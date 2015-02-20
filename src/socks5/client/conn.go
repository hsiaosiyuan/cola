package client

import (
	"net"
	"bufio"
	"errors"
	"bytes"
	"sync"
	"io"
	"time"
)

var (
	ErrDialSrvFailed = errors.New("Dial Server: Failed to dial to server.")

	ErrSMSWriteBytes = errors.New("Send method selection: Failed to send method selection.")

	ErrApplySubNegCannotReadMethodBytes = errors.New("Apply Sub-negotiate: Failed to read method bytes.")
	ErrApplySubNegInvalidVerNum         = errors.New("Apply Sub-negotiate: not supported version.")
	ErrApplySubNegNoneMatch             = errors.New("Apply Sub-negotiate: not supported method.")

	ErrNegConnReadBytes     = errors.New("NegConn: Failed to read bytes.")
	ErrNegConnInvalidVerNum = errors.New("NegConn: Invalid version number.")
	ErrNegConnMethodNone    = errors.New("NegConn: No supported method.")
	ErrNegConnWriteReplay   = errors.New("NegConn: Failed to write replay.")

	ErrNegSrvReadBytes     = errors.New("NegSrv: Failed to read bytes.")
	ErrNegSrvInvalidVerNum = errors.New("NegSrv: Invalid version number.")
	ErrNegSrvNoUnPwd       = errors.New("NegSrv: No UnPwd supported.")

	ErrNegSrvAuthSend          = errors.New("NegSrvAuth: Failed to send bytes.")
	ErrNegSrvAuthReadBytes     = errors.New("NegSrvAuth: Failed to read bytes.")
	ErrNegSrvAuthInvalidVerNum = errors.New("NegSrvAuth: Invalid version number.")
	ErrNegSrvAuthInvalid       = errors.New("NegSrvAuth: Invalid username or password.")
)

type Conn struct {
	Client *Client
	Conn net.Conn
	ConnBr *bufio.Reader
	SrvConn *net.TCPConn
	SrvBr *bufio.Reader
}

func (c *Conn) Serve() {
	var (
		err error
	)

	if err = c.negConn(); err != nil {
		c.Client.Log(err)
		c.Close()
		return
	}

	if err = c.negWithSrv(); err != nil {
		c.Client.Log(err)
		c.Close()
		return
	}

	if err = c.exchange(); err != nil {
		c.Client.Log(err)
	}

	c.Close()
}

func (c *Conn) negConn() error {
	var (
		buf = make([]byte, 257)
		err error
		methodCount uint8
		methods []byte
		rep []byte
	)

	if _, err = c.ConnBr.Read(buf); err != nil {
		return ErrNegConnReadBytes
	}

	if buf[0] != Version5 {
		return ErrNegConnInvalidVerNum
	}

	methodCount = uint8(buf[1])
	methods = buf[2:methodCount+2]
	if bytes.IndexByte(methods, MethodBare) == -1 {
		return ErrNegConnMethodNone
	}

	rep = []byte{Version5, MethodBare}
	if _, err = c.Conn.Write(rep); err != nil {
		return ErrNegConnWriteReplay
	}

	return nil
}

func (c *Conn) dialServer() error {
	var (
		err error
		t time.Time
	)

	if c.SrvConn, err = net.DialTCP("tcp", nil, c.Client.SrvAddr); err != nil {
		c.Client.Log(err)
		return ErrDialSrvFailed
	}

	t = time.Now().Add(time.Minute*2)
	c.SrvConn.SetDeadline(t)

	c.SrvBr = bufio.NewReader(c.SrvConn)
	return nil
}

// send method selection to server
func (c *Conn) sendMsToSrv() error {
	var (
		err error
		selection []byte
	)

	selection = []byte{Version5, 1, MethodUnPwd}
	if _, err = c.SrvConn.Write(selection); err != nil {
		return ErrSMSWriteBytes
	}

	return nil
}

// negotiate with server by using "username/password" auth method
func (c *Conn) negWithSrv() error {
	var (
		err error
		buf = make([]byte, 2)
		subNegB []byte
	)

	if err = c.dialServer(); err != nil {
		return err
	}

	if err = c.sendMsToSrv(); err != nil {
		return err
	}

	if _, err = c.SrvBr.Read(buf); err != nil {
		return ErrNegSrvReadBytes
	}

	if buf[0] != Version5 {
		return ErrNegSrvInvalidVerNum
	}

	if buf[1] != MethodUnPwd {
		return ErrNegSrvNoUnPwd
	}

	subNegB = []byte{MethodUnPwdVer, byte(len(c.Client.Un))}
	subNegB = append(subNegB, c.Client.Un...)
	subNegB = append(subNegB, byte(len(c.Client.Pwd)))
	subNegB = append(subNegB, c.Client.Pwd...)

	if _, err = c.SrvConn.Write(subNegB); err != nil {
		return ErrNegSrvAuthSend
	}

	if _, err = c.SrvBr.Read(buf); err != nil {
		return ErrNegSrvAuthReadBytes
	}

	if buf[0] != MethodUnPwdVer {
		return ErrNegSrvAuthInvalidVerNum
	}

	if buf[1] != MethodUnPwdStatusOk {
		return ErrNegSrvAuthInvalid
	}

	return nil
}

func (c *Conn) exchange() error {
	var (
		wg sync.WaitGroup
		errL2R error
		errR2L error
	)

	wg.Add(2)

	go func() {
		_, errL2R = io.Copy(c.SrvConn, c.Conn)
		wg.Done()
	}()

	go func() {
		_, errR2L = io.Copy(c.Conn, c.SrvConn)
		wg.Done()
	}()

	wg.Wait()

	if errL2R != nil {
		return errL2R
	}

	if errR2L != nil {
		return errR2L
	}

	return nil
}

func (c *Conn) Close() {
	if c.Conn != nil {
		c.Conn.Close();
		c.Conn = nil
	}

	if c.SrvConn != nil {
		c.SrvConn.Close()
		c.SrvConn = nil
	}

	c.Client.Log("connect closed.")
}
