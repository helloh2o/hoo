package hox

import (
	"bufio"
	"fmt"
	"golang.org/x/time/rate"
	"io"
	"log"
	"net"
	"time"
)

type conn struct {
	user       string
	rwc        net.Conn
	remote     net.Conn
	brc        *bufio.Reader
	server     *Server
	maxSpeed   rate.Limit
	TotalRead  int64
	TotalWrite int64
}

func (c *conn) auth(credential string) bool {
	if c.server.isAuth() == false {
		return true
	} else {
		ok, user := c.server.validateAuth(credential)
		if ok {
			c.user = user
			// verify traffic
			tr, ok := clientsTraffic.Load(c.user)
			if !ok {
				onConnecting <- c.user
				return true
			} else {
				v, _ := tr.(int64)
				if v > 0 {
					return true
				} else {
					c.rwc.Close()
					log.Printf("Close traffic use up user %s", c.user)
					return false
				}
			}

		}
	}
	return false
}
func (c *conn) serve() {
	defer c.rwc.Close()
	rawHttpHeader, remote, credential, connect, err := parseReq(c, c.server.Free)
	if err != nil {
		c.rwc.Write([]byte("HTTP/1.1 503 Service Unavailable\r\n\r\n"))
		return
	}
	remoteConn, err := net.DialTimeout("tcp", remote, time.Second*5)
	if err != nil {
		return
	}
	defer func() {
		// remove & close
		remoteConn.Close()
		// record
		total := c.TotalWrite + c.TotalRead
		if record, ok := records.Load(c.user); ok {
			usedBytes, _ := record.(int64)
			total += usedBytes
		}
		records.Store(c.user, total)
	}()
	if credential == "" || c.auth(credential) == false {
		msg503 := "HTTP/1.1 503 service unavailable\r\n\r\n"
		_, err := c.rwc.Write([]byte(msg503))
		if err != nil {
			return
		}
	}
	/*if c.auth(credential) == false {
		// Require auth
		var respBf bytes.Buffer
		respBf.WriteString("HTTP/1.1 407 Proxy Authentication Required\r\n")
		respBf.WriteString("Proxy-Authenticate: Basic realm=\"hox\"\r\n")
		respBf.WriteString("\r\n")
		respBf.WriteTo(c.rwc)
		return
	}*/
	if connect {
		msg200 := "HTTP/1.1 200 Connection established\r\n\r\n"
		// if connect, send 200
		_, err := c.rwc.Write([]byte(msg200))
		if err != nil {
			return
		}
	} else {
		if _, err := rawHttpHeader.WriteTo(remoteConn); err != nil {
			fmt.Println(err)
			return
		}
	}
	// build tunnel
	c.remote = remoteConn
	c.tunnel()
}

func (c *conn) tunnel() {
	client := c.rwc
	src := c.remote
	defer func() {
		//log.Println("===============tunnel serve goroutine done================")
	}()
	bufClient := make([]byte, 1024)
	go func() {
		defer func() {
			src.Close()
			//log.Println("===============client goroutine done================")
		}()
		for {
			n, er := client.Read(bufClient)
			if er != nil && n == 0 {
				break
			}
			if n > 0 {
				wn, ew := src.Write(bufClient[:n])
				if ew != nil || wn != n {
					//fmt.Println("------------- client connection write to error -------------")
					break
				}
				c.TotalRead += int64(wn)
			}
		}
	}()
	c.pipe(src)
}

func (c *conn) pipe(src net.Conn) {
	var remoteReader io.Reader
	var buf []byte
	writen := 0
	mb5 := 1024 * 1024 * 5
	remoteReader = src
	buf = make([]byte, 32*1024)
	limited := false
	for {
		nr, er := remoteReader.Read(buf)
		if er != nil && nr == 0 {
			break
		}
		if nr > 0 {
			nw, ew := c.rwc.Write(buf[0:nr])
			if ew != nil {
				break
			} else {
				// limit speed > 5MB
				if !limited && writen > mb5 && c.maxSpeed > 0 {
					limit := rate.NewLimiter(c.maxSpeed*1024, int(c.maxSpeed)*1024)
					remoteReader = NewReader(src, limit)
					limited = true
				}
				writen += nw
				c.TotalWrite += int64(nw)
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
