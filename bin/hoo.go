package main

import (
	"flag"
	"fmt"
	"hoo"
	"io/ioutil"
	"log"
	"os"
	"time"
)

var (
	addr = flag.String("addr", ":31280", "listen addr")
	cert = flag.String("c", "", "cert file")
	key  = flag.String("k", "", "cert key file")
	host = flag.String("host", "", "host name")
	auth = flag.String("auth", "admin:admin", "eg-> name:pass")
	max  = flag.Float64("max", 720, "max speed of connection (KB/s)")
	free = flag.Bool("free", false, "free server or not")
)

func main() {
	flag.Parse()
	hoo.SyncInit()
	if *host != "" {
		*cert = fmt.Sprintf("/root/.caddy/acme/acme-v02.api.letsencrypt.org/sites/%s/%s.crt", *host, *host)
		*key = fmt.Sprintf("/root/.caddy/acme/acme-v02.api.letsencrypt.org/sites/%s/%s.key", *host, *host)
	}
	// check ssl
	go func(certPath, keyPath string) {
		var tempCertString, tempKeyString string
		tk := time.Tick(time.Hour)
		for {
			cfile, err := os.Open(certPath)
			if err != nil {
				return
			}
			kfile, err := os.Open(keyPath)
			if err != nil {
				return
			}
			cf, err := ioutil.ReadAll(cfile)
			if err != nil {
				return
			}
			kf, err := ioutil.ReadAll(kfile)
			if err != nil {
				return
			}
			newc := string(cf)
			newk := string(kf)
			if tempCertString != "" && tempKeyString != "" && (newc != tempCertString || newk != tempKeyString) {
				// ssl changed
				log.Printf("Change SSL file \noldKey %s \nnewKey %s\noldCert %s\nnewCert %s", tempKeyString, newk, tempCertString, newc)
				os.Exit(-1)
			} else {
				tempCertString = newc
				tempKeyString = newk
				cfile.Close()
				kfile.Close()
				<-tk
			}
		}
	}(*cert, *key)
	s := hox.NewServer(*addr, *auth, *cert, *key, *max, *free)
	s.Start()
}
