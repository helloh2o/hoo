package hox

import (
	"sync"
	"time"
	"fmt"
)

type Pool interface {
	Put(string, *conn) bool
	Remove(string, *conn)
	Clean(string) bool
	Manage()
}

type RmtPool struct {
	sync.Mutex
	queueMap map[string][]*conn
	maxIdle  int
}

var pool Pool

func init() {
	p := new(RmtPool)
	p.maxIdle = 1000
	p.queueMap = make(map[string][]*conn)
	pool = p
	go p.Manage()
}
func (rp *RmtPool) Manage() {
	tk := time.NewTimer(time.Minute * 5)
	for {
		<-tk.C
		fmt.Println("=============== timer check begin =============== ")
		rp.Lock()
		for ip, hub := range rp.queueMap {
			check(ip, hub)
		}
		rp.Unlock()
		fmt.Println("=============== timer check end =============== ")
	}
}

func check(ip string, hub []*conn) {
	now := time.Now().Unix()
	tenMin := int64(300)
	for i := 0; i < len(hub); i++ {
		c := hub[i]
		// idle ten minutes
		if c.LastWrite+tenMin < now && c.LastRead+tenMin < now {
			c.rwc.Close()
			c.remote.Close()
			/*// remove from queue
			hub = append(hub[:i], hub[i+1:]...)*/
			fmt.Printf("ip %s, conn closed -> idle timeout.\n", ip)
			// maintain the correct index
			//i--
		}
	}
}

func (rp *RmtPool) Clean(ip string) bool {
	defer rp.Unlock()
	rp.Lock()
	hub := rp.queueMap[ip]
	check(ip, hub)
	return len(hub) < rp.maxIdle
}

func (rp *RmtPool) Put(ip string, c *conn) bool {
	defer rp.Unlock()
	rp.Lock()
	if hub, ok := rp.queueMap[ip]; ok {
		ltop := len(hub)
		if ltop > rp.maxIdle {
			return false
		}
		hub = append(hub, c)
		rp.queueMap[ip] = hub
		fmt.Printf("ip %s conn hub length %d \n", ip, ltop+1)
	} else {
		hub = append([]*conn{}, c)
		rp.queueMap[ip] = hub
	}
	return true
}

func (rp *RmtPool) Remove(ip string, rc *conn) {
	defer rp.Unlock()
	rp.Lock()
	if hub, ok := rp.queueMap[ip]; ok {
		for i, c := range hub {
			if rc == c {
				hub = append(hub[:i], hub[i+1:]...)
				rp.queueMap[ip] = hub
				break
			}
		}
		fmt.Printf("Remove from hub , len %d\n", len(hub))
	}
}
