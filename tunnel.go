package tunnel

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"time"
	"sync"
	"strings"
)

type backend struct {
	Host     string
	Port     string
	Hostname string
	TLS      bool
	Insecure bool
}

var (
	dialer     = &net.Dialer{Timeout: 6 * time.Second}
	l          net.Listener
	host, port string
	nodes      = make(map[string]string)
	running    = false
)

func Run(remote, local string) {
	if running {
		return
	}
	go initListener(remote, local)
	select {}
}
func SwitchNode(remote string) bool {
	return true
}

func TestWss(nodesString string) string {
	nodes := strings.Split(nodesString, "|")
	ret := make(chan string, 100)
	var wg sync.WaitGroup
	wg.Add(len(nodes))
	for i := 0; i < len(nodes); i++ {
		go func(x int) {
			r := nodes[x]
			// dial
			h, p, _ := net.SplitHostPort(r)
			config := &tls.Config{
				ServerName:         h,
				InsecureSkipVerify: false,
			}
			_, err := tls.DialWithDialer(dialer, "tcp", h+":"+p, config)
			if err == nil {
				ret <- r
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	log.Printf("test over, ok size %d\n", len(ret))
	result := ""
	for {
		if len(ret) > 0 {
			per := <-ret
			if result == "" {
				result += per
			} else {
				result += "|" + per
			}
		} else {
			break
		}
	}
	log.Printf("WSS RET %s \n", result)
	return result
}


func initListener(remote, local string) {
	var err error
	host, port, err = net.SplitHostPort(remote)
	if err != nil {
		log.Fatal(err)
	}
	bk := backend{Port: port, Host: host, Hostname: host, TLS: true, Insecure: false}
	l, err = net.Listen("tcp", local)
	if err != nil {
		log.Printf("Listen error :: %v\n", err)
		return
	} else {
		log.Printf("Listen on %s\n", local)
		running = true
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println(err)
			l.Close()
			break
		}
		go handleConn(conn, bk)
	}
	running = false
}

func handleConn(conn net.Conn, b backend) {
	var c net.Conn
	var err error

	remote := net.JoinHostPort(host, port)
	if b.TLS {
		config := &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: false,
		}
		c, err = tls.DialWithDialer(dialer, "tcp", remote, config)
	} else {
		c, err = dialer.Dial("tcp", host)
	}

	if err != nil {
		log.Println(err)
		conn.Close()
		return
	}

	pipeAndClose(conn, c)
}

func pipeAndClose(c1, c2 net.Conn) {
	defer c1.Close()
	defer c2.Close()

	ch := make(chan struct{}, 2)
	go func() {
		io.Copy(c1, c2)
		ch <- struct{}{}
	}()

	go func() {
		io.Copy(c2, c1)
		ch <- struct{}{}
	}()
	<-ch
}
