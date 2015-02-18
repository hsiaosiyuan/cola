package socks5

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
)

const (
	version5 = byte(5)

	authMethodNone               = byte(0xFF)
	authMethodBare               = byte(0)
	authMethodUnPwd              = byte(2)
	authMethodUnPwdStatusSucceed = byte(0)
	authMethodUnPwdVersion       = byte(1)

	cmdConnect = byte(1)
	reqRsv     = byte(0)
	aTypIpv4   = byte(1)
	aTypDomain = byte(3)
	aTypIpv6   = byte(4)

	repSucceeded            = byte(0)
	repGeneralServerFailure = byte(1)
	repNetUnReachable       = byte(3)
	repHostUnReachable      = byte(4)
	repConnRefused          = byte(5)
	repCmdNotSupported      = byte(7)
	repATypNotSupported     = byte(8)
)

var (
	ErrNegotiateReadBytes             = errors.New("Negotiate: failed to read negotiate bytes.")
	ErrNegotiateNotSupportedVersion   = errors.New("Negotiate: unsupported version.")
	ErrNegotiateInvalidAuthMethodsNum = errors.New("Negotiate: num of auth methods was zero or greater than 255.")
	ErrNegotiateNoSupportedAuthMethod = errors.New("Negotiate: all auth methods were unsupported.")
	ErrNegotiateWriteReplay           = errors.New("Negotiate: failed to write repaly.")

	ErrAuthBareFailedToWriteReplay  = errors.New("Bare sub-negotiate: failed to write replay.")
	ErrAuthUnPwdFailedToWriteReplay = errors.New("UnPwd sub-negotiate: failed to write replay.")
	ErrAuthUnPwdFailedToReadUnPwd   = errors.New("UnPwd sub-negotiate: failed to read UnPwd bytes.")
	ErrAuthUnPwdNotSupportedVersion = errors.New("UnPwd sub-negotiate: invalid version.")
	ErrAuthUnPwdInvalidUnLength     = errors.New("UnPwd sub-negotiate: invalid username length.")
	ErrAuthUnPwdInvalidPwdLength    = errors.New("UnPwd sub-negotiate: invalid password length.")
	ErrAuthUnPwdInvalidUnOrPwd      = errors.New("UnPwd sub-negotiate: invalid username or password.")
	ErrAuthUnPwdWriteReplay         = errors.New("UnPwd sub-negotiate: failed to write replay.")

	ErrParseCmdReadBytes          = errors.New("ParseCmd: failed to read bytes.")
	ErrParseCmdUnsupportedVersion = errors.New("ParseCmd: unsupported version.")
	ErrParseCmdUnsupportedCmd     = errors.New("ParseCmd: unsupported command.")
	ErrParseCmdInvalidRsv         = errors.New("ParseCmd: invalid RSV.")
	ErrParseCmdInvalidATyp        = errors.New("ParseCmd: invalid address type.")

	ErrParseDstAddrATypIpv4ReadBytes   = errors.New("ParseDstAddr: failed to read IPV4 bytes.")
	ErrParseDstAddrATypDomainReadBytes = errors.New("ParseDstAddr: failed to read domain bytes.")
	ErrParseDstAddrATypIpv6ReadBytes   = errors.New("ParseDstAddr: failed to read IPV6 bytes.")
	ErrParseDstAddrInvalid             = errors.New("ParseDstAddr: invalid address")

	ErrPrepareExchangeGeneral          = errors.New("PrepareExchange: general error.")
	ErrPrepareExchangeATypNotSupported = errors.New("PrepareExchange: unsupport address type.")
	ErrPrepareExchangeHostUnReachable  = errors.New("PrepareExchange: host unreachable.")
	ErrPrepareExchangeNetUnReachable   = errors.New("PrepareExchange: network unreachable.")
	ErrPrepareExchangeConnRefused      = errors.New("PrepareExchange: connect refused.")

	ErrExchangeL2R = errors.New("Exchange: left to right.")
	ErrExchangeR2L = errors.New("Exchange: right to left.")

	ErrWriteCmdReplay = errors.New("CmdReplay: failed to write data.")
)

type conn struct {
	netConn net.Conn
	server  *Server
	br      *bufio.Reader
	method  byte
	aTyp    byte
	dstHost string
	dstAddr *net.TCPAddr
	dstConn *net.TCPConn
}

func (c *conn) serve() {
	var (
		err error
	)

	if err = c.negotiate(); err != nil {
		c.server.Log(err)
		c.close()
	}

	if err = c.exchange(); err != nil {
		c.server.Log(err)
		c.close()
	}
}

func (c *conn) negotiate() error {
	var (
		buf        = make([]byte, 257)
		err        error
		methodsNum uint8
		methods    []byte
	)

	if _, err = c.br.Read(buf); err != nil {
		return ErrNegotiateReadBytes
	}

	if buf[0] != version5 {
		return ErrNegotiateNotSupportedVersion
	}

	if methodsNum = uint8(buf[1]); methodsNum == 0 || methodsNum > 255 {
		return ErrNegotiateInvalidAuthMethodsNum
	}

	methods = buf[2 : methodsNum+2]
	if bytes.IndexByte(methods, authMethodUnPwd) != -1 {
		if err = c.writeNegotiateReplay(authMethodUnPwd); err != nil {
			return err
		}

		return c.subNegotiateAuthUnPwd()
	}

	if bytes.IndexByte(methods, authMethodBare) != -1 {
		return c.writeNegotiateReplay(authMethodBare)
	}

	if err = c.writeNegotiateReplay(authMethodNone); err != nil {
		return err
	}

	return ErrNegotiateNoSupportedAuthMethod
}

func (c *conn) subNegotiateAuthUnPwd() error {
	var (
		buf  = make([]byte, 513)
		err  error
		uLen uint8
		pLen uint8
		un   []byte
		pwd  []byte
		ok   bool
	)

	c.method = authMethodUnPwd

	if _, err = c.br.Read(buf); err != nil {
		return ErrAuthUnPwdFailedToReadUnPwd
	}

	if buf[0] != authMethodUnPwdVersion {
		return ErrAuthUnPwdNotSupportedVersion
	}

	if uLen = uint8(buf[1]); uLen == 0 || uLen > 255 {
		return ErrAuthUnPwdInvalidUnLength
	}

	un = buf[2 : uLen+2]

	if pLen = uint8(buf[uLen+2]); pLen == 0 || pLen > 255 {
		return ErrAuthUnPwdInvalidPwdLength
	}

	pwd = buf[3+uLen : 3+uLen+pLen]
	ok = c.server.Cfg.AuthUnPwd(string(un), string(pwd))

	return c.writeAuthUnPwdReplay(ok)
}

func (c *conn) parseCommand() error {
	var (
		buf []byte
		err error
	)

	buf = make([]byte, 4)
	if _, err = c.br.Read(buf); err != nil {
		return ErrParseCmdReadBytes
	}

	if buf[0] != version5 {
		return ErrParseCmdUnsupportedVersion
	}

	if buf[1] != cmdConnect {
		return ErrParseCmdUnsupportedCmd
	}

	if buf[2] != reqRsv {
		return ErrParseCmdInvalidRsv
	}

	c.aTyp = buf[3]
	switch c.aTyp {
	case aTypIpv4, aTypDomain, aTypIpv6:
		return c.parseDstAddr()
	default:
		return ErrParseCmdInvalidATyp
	}

	return nil
}

func (c *conn) parseDstAddr() error {
	var (
		err       error
		buf       []byte
		addr      []byte
		portBytes = make([]byte, 2)
		port      uint16
		dLenByte  byte
		dLen      uint8
		host      string
	)

	switch c.aTyp {
	case aTypIpv4:
		addr = make([]byte, 4)
		if _, err = c.br.Read(addr); err != nil {
			return ErrParseDstAddrATypIpv4ReadBytes
		}

		if _, err = c.br.Read(portBytes); err != nil {
			return ErrParseDstAddrATypIpv4ReadBytes
		}

		host = net.IPv4(addr[0], addr[1], addr[2], addr[3]).String()
	case aTypDomain:
		if dLenByte, err = c.br.ReadByte(); err != nil {
			return ErrParseDstAddrATypDomainReadBytes
		}

		dLen = uint8(dLenByte)
		buf = make([]byte, dLen)
		if _, err = c.br.Read(buf); err != nil {
			return ErrParseDstAddrATypDomainReadBytes
		}

		if _, err = c.br.Read(portBytes); err != nil {
			return ErrParseDstAddrATypDomainReadBytes
		}

		host = string(buf)
	case aTypIpv6:
		buf = make([]byte, 16)
		if _, err = c.br.Read(buf); err != nil {
			return ErrParseDstAddrATypIpv6ReadBytes
		}

		if _, err = c.br.Read(portBytes); err != nil {
			return ErrParseDstAddrATypIpv6ReadBytes
		}

		host = net.IP(buf).String()
	}

	port = uint16(portBytes[0])<<8 + uint16(portBytes[1])
	c.dstHost = host + ":" + strconv.Itoa(int(port))
	if c.dstAddr, err = net.ResolveTCPAddr("tcp", c.dstHost); err != nil {
		return ErrParseDstAddrInvalid
	}

	return nil
}

func (c *conn) prepareExchange() error {
	var (
		err error
	)

	if c.dstConn, err = net.DialTCP("tcp", nil, c.dstAddr); err != nil {
		switch e := err.(type) {
		case *net.OpError:
			switch e.Err.(type) {
			case *net.AddrError:
				return ErrPrepareExchangeATypNotSupported
			case *net.DNSError:
				return ErrPrepareExchangeHostUnReachable
			default:
				if e.Err.Error() == "network is unreachable" {
					return ErrPrepareExchangeNetUnReachable
				}

				if e.Err.Error() == "connection refused" {
					return ErrPrepareExchangeConnRefused
				}

				return err
			}
		default:
			return ErrPrepareExchangeGeneral
		}
	}

	return nil
}

func (c *conn) exchange() error {
	var (
		wg     sync.WaitGroup
		errL2R error
		errR2L error
		err    error
	)

	if err = c.parseCommand(); err != nil {
		c.server.Log(err)

		switch err {
		case ErrParseCmdUnsupportedVersion,
		ErrParseCmdUnsupportedCmd,
		ErrParseCmdInvalidRsv:
			return c.writeCmdReplay(repCmdNotSupported)
		case ErrParseCmdInvalidATyp, ErrParseDstAddrInvalid:
			return c.writeCmdReplay(repATypNotSupported)
		default:
			return err
		}
	}

	if err = c.prepareExchange(); err != nil {
		c.server.Log(err)

		switch err {
		case ErrPrepareExchangeATypNotSupported:
			return c.writeCmdReplay(repATypNotSupported)
		case ErrPrepareExchangeHostUnReachable:
			return c.writeCmdReplay(repHostUnReachable)
		case ErrPrepareExchangeNetUnReachable:
			return c.writeCmdReplay(repNetUnReachable)
		case ErrPrepareExchangeConnRefused:
			return c.writeCmdReplay(repConnRefused)
		default:
			return c.writeCmdReplay(repGeneralServerFailure)
		}
	}

	c.writeCmdReplay(repSucceeded)

	wg.Add(2)

	go func() {
		_, errL2R = io.Copy(c.dstConn, c.netConn)
		wg.Done()
	}()

	go func() {
		_, errR2L = io.Copy(c.netConn, c.dstConn)
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

func (c *conn) writeCmdReplay(repField byte) error {
	var (
		aTyp    byte
		bndAddr []byte
		bndPort []byte
		lAddr   *net.TCPAddr
		err     error
		rep     []byte
		port16  uint16
	)

	rep = []byte{version5, repField, reqRsv}

	if c.dstConn != nil {
		lAddr = c.dstConn.LocalAddr().(*net.TCPAddr)
		if lAddr.Zone == "" {
			aTyp = aTypIpv4
		} else {
			aTyp = aTypIpv6
		}

		rep = append(rep, aTyp)

		bndAddr = []byte(lAddr.IP)
		rep = append(rep, bndAddr...)

		port16 = uint16(lAddr.Port)
		bndPort = make([]byte, 2)
		bndPort[0] = byte(port16)
		bndPort[1] = byte(port16 >> 8)
		rep = append(rep, bndPort...)
	}

	if _, err = c.netConn.Write(rep); err != nil {
		return ErrWriteCmdReplay
	}

	if repField != repSucceeded {
		c.close()
	}

	return nil
}

func (c *conn) writeNegotiateReplay(method byte) error {
	if _, err := c.netConn.Write([]byte{version5, method}); err != nil {
		return ErrNegotiateWriteReplay
	}

	return nil
}

func (c *conn) writeAuthUnPwdReplay(ok bool) error {
	var (
		status byte
		err    error
	)

	if ok {
		status = authMethodUnPwdStatusSucceed
	} else {
		status = byte(1)
	}

	if _, err = c.netConn.Write([]byte{authMethodUnPwdVersion, status}); err != nil {
		return ErrAuthUnPwdWriteReplay
	}

	return nil
}

func (c *conn) close() {
	if c.dstConn != nil {
		c.dstConn.Close()
		c.dstConn = nil
	}

	if c.netConn != nil {
		c.netConn.Close()
		c.netConn = nil
	}
}
