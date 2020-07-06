package hox

import (
	"context"
	"flag"
	"github.com/smallnest/rpcx/client"
	"log"
	"sync"
	"time"
)

var (
	records = sync.Map{}
	tk      = time.Tick(time.Minute)
	rpc            = flag.String("rpc", "127.0.0.1:2020", "sync rpc address")
	clientsTraffic = sync.Map{}
	onConnecting   chan string
)

type Args struct {
	User    string
	Traffic int64 //mb
}

func SyncInit() {
	onConnecting = make(chan string)
	go func() {
		for {
			d := client.NewPeer2PeerDiscovery("tcp@"+*rpc, "")
			xclient := client.NewXClient("TR", client.Failtry, client.RandomSelect, d, client.DefaultOption)
			defer xclient.Close()
			select {
			case <-tk:
				records.Range(func(user, record interface{}) bool {
					mb := 1024 * 1024
					usedBytes, _ := record.(int64)
					used := usedBytes / int64(mb)
					needUpdate := false
					uuid, _ := user.(string)
					// traffic > 1mb
					if used > 0 {
						needUpdate = true
					}
					// no user records
					if _, ok := clientsTraffic.Load(uuid); !ok {
						needUpdate = true
					}
					if needUpdate {
						err := syncTr(xclient, uuid, used)
						if err != nil {
							return false
						}
					}
					return true
				})
			case uuid := <-onConnecting:
				syncTr(xclient, uuid, 0)
			}
		}
	}()
}

func syncTr(xclient client.XClient, uuid string, used int64) error {
	//sync traffic
	args := Args{
		User:    uuid,
		Traffic: used,
	}
	var reply int64
	err := xclient.Call(context.Background(), "Sync", &args, &reply)
	if err != nil {
		log.Printf("sync traffic end, rpc error %v", err)
		return err
	} else {
		// update succeed
		records.Store(args.User, 0)
		clientsTraffic.Store(args.User, reply)
	}
	return nil
}
