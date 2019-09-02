package hox

import (
	"net"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"golang.org/x/time/rate"
	"time"
	"log"
)

type conn struct {
	rwc      net.Conn
	brc      *bufio.Reader
	server   *Server
	maxSpeed rate.Limit
}

func (c *conn) auth(credential string) bool {
	if c.server.isAuth() == false || c.server.validateAuth(credential) {
		return true
	}
	return false
}
func (c *conn) handle() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("panic recover connection serve ::", r)
		}
		c.rwc.Close()
	}()
	rawHttpHeader, remote, credential, connect, err := parseReq(c.brc)
	if err != nil {
		fmt.Println(err)
		c.rwc.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\n"))
		c.rwc.Close()
		return
	}
	remoteConn, err := net.DialTimeout("tcp", remote, time.Second*3)
	if err != nil {
		return
	}
	defer remoteConn.Close()
	if err != nil {
		fmt.Println("getTunnelInfo -> ", err)
		return
	}
	if c.auth(credential) == false {
		// Require auth
		var respBf bytes.Buffer
		respBf.WriteString("HTTP/1.1 407 Proxy Authentication Required\r\n")
		respBf.WriteString("Proxy-Authenticate: Basic realm=\"hox\"\r\n")
		respBf.WriteString("\r\n")
		respBf.WriteTo(c.rwc)
		return
	}

	if connect {
		// if connect, send 200
		_, err := c.rwc.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
		if err != nil {
			fmt.Println("Connection established write error ->", err)
			return
		}
	} else {
		if _, err := rawHttpHeader.WriteTo(remoteConn); err != nil {
			fmt.Println(err)
			return
		}
		log.Println("=============== HTTP =====================")
		c.pipe(remoteConn)
		log.Println("=============== HTTP DONE =====================")
		return
	}
	// build tunnel
	fmt.Println("tunnel,", c.rwc.RemoteAddr(), "<->", remote)
	c.tunnel(remoteConn)
	//pool.Put(remote, remoteConn)
}

func (c *conn) tunnel(remote net.Conn) {
	defer func() {
		// pull to pool
		log.Println("===============tunnel goroutine done================")
	}()
	client := c.rwc
	src := remote
	bufClient := make([]byte, 1024)
	go func() {
		defer log.Println("===============client goroutine done================")
		for {
			n, er := client.Read(bufClient)
			if er != nil && n == 0{
				break
			}
			if n > 0 {
				wn, ew := src.Write(bufClient[:n])
				if ew != nil || wn != n {
					fmt.Println("------------- client connection write to error -------------")
					break
				}
			}
		}
	}()
	c.pipe(src)
}

func (c *conn) pipe(src net.Conn) {
	var remoteReader io.Reader
	var buf []byte
	writen := 0
	ten := 1024 * 1024 * 10
	remoteReader = src
	buf = make([]byte, 32*1024)
	limited := false
	for {
		nr, er := remoteReader.Read(buf)
		if er != nil  && nr == 0 {
			break
		}
		if nr > 0 {
			nw, ew := c.rwc.Write(buf[0:nr])
			if ew != nil {
				break
			} else {
				// limit speed > 10MB
				if  !limited && writen > ten && c.maxSpeed > 0{
					limit := rate.NewLimiter(c.maxSpeed*1024, int(c.maxSpeed)*1024)
					remoteReader = NewReader(src, limit)
					limited = true
				} else {
					writen += nw
				}
			}
			if nr != nw {
				break
			}
		}
	}
}

type BadRequestError struct {
	what string
}

func (b *BadRequestError) Error() string {
	return b.what
}
