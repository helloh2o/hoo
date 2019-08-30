package hox
/*
import (
	"net"
	"sync"
	"log"
)

type Pool interface {
	Get(string) net.Conn
	Put(string, net.Conn)
	Reset()
}

type RmtPool struct {
	sync.Mutex
	queueMap map[string]chan net.Conn
}

var pool Pool

func init() {
	p := new(RmtPool)
	p.Reset()
	pool = p
}

func (rp *RmtPool) Reset() {
	rp.queueMap = make(map[string]chan net.Conn)
}

func (rp *RmtPool) Get(host string) net.Conn {
	defer rp.Unlock()
	rp.Lock()
	if hub, ok := rp.queueMap[host]; ok {
		select {
		case c := <-hub:
			log.Printf("----Get pool conn for host %s----\n", host)
			return c
		default:
			return nil
		}
	}
	return nil
}

func (rp *RmtPool) Put(host string, c net.Conn) {
	defer rp.Unlock()
	rp.Lock()
	if hub, ok := rp.queueMap[host]; ok {
		select {
		case hub <- c:
		default:
			log.Printf("Queue for host '%s'is full !!  %d \n", host, len(hub))
			return
		}
	} else {
		rp.queueMap[host] = make(chan net.Conn, 100)
		rp.queueMap[host] <- c
	}
}
*/
