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
	"sync"
)

type conn struct {
	LK        sync.Mutex
	rwc       net.Conn
	remote    net.Conn
	brc       *bufio.Reader
	server    *Server
	maxSpeed  rate.Limit
	LastRead  int64
	LastWrite int64
}

func (c *conn) lastRead() {
	c.LK.Lock()
	defer c.LK.Unlock()
	c.LastRead = time.Now().Unix()
}

func (c *conn) lastWrite() {
	c.LK.Lock()
	defer c.LK.Unlock()
	c.LastWrite = time.Now().Unix()
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
		return
	}
	remoteConn, err := net.DialTimeout("tcp", remote, time.Second*3)
	ip, _, _ := net.SplitHostPort(c.rwc.RemoteAddr().String())
	if err != nil {
		return
	}
	defer func() {
		// remove & close
		pool.Remove(ip, c)
		remoteConn.Close()
	}()
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
		msg200 := "HTTP/1.1 200 Connection established\r\n\r\n"
		if ok := pool.Put(ip, c); ok {
			// if connect, send 200
			_, err := c.rwc.Write([]byte(msg200))
			if err != nil {
				return
			}
		} else if ok := pool.Clean(ip); ok {
			if ok = pool.Put(ip, c); ok {
				// if connect, send 200
				_, err := c.rwc.Write([]byte(msg200))
				if err != nil {
					return
				}
			}
		} else {
			c.rwc.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\n"))
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
	c.remote = remoteConn
	c.tunnel()
}

func (c *conn) tunnel() {
	defer func() {
		// pull to pool
		log.Println("===============tunnel goroutine done================")
	}()
	client := c.rwc
	src := c.remote
	bufClient := make([]byte, 1024)
	go func() {
		defer log.Println("===============client goroutine done================")
		for {
			n, er := client.Read(bufClient)
			if er != nil {
				break
			}
			if n > 0 {
				wn, ew := src.Write(bufClient[:n])
				if ew != nil || wn != n {
					fmt.Println("------------- client connection write to error -------------")
					break
				}
				c.LastRead = time.Now().Unix()
			}
		}
	}()
	c.pipe(src)
}

func (c *conn) pipe(src net.Conn) {
	var remoteReader io.Reader
	var buf []byte
	writen := 0
	ten := 1024 * 1024 * 5
	remoteReader = src
	buf = make([]byte, 32*1024)
	for {
		nr, er := remoteReader.Read(buf)
		if er != nil {
			break
		}
		if nr > 0 {
			nw, ew := c.rwc.Write(buf[0:nr])
			if ew != nil {
				break
			} else {
				// limit speed > 10MB
				if writen > ten && c.maxSpeed > 0 {
					limit := rate.NewLimiter(c.maxSpeed*1024, int(c.maxSpeed)*1024)
					remoteReader = NewReader(src, limit)
					writen = 0
				} else {
					writen += nw
				}
				c.LastWrite = time.Now().Unix()
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
