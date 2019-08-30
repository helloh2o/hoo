package main

import (
	_ "net/http/pprof"
	"flag"
	"fmt"
	"hox"
)

var (
	addr = flag.String("addr", ":31280", "listen addr")
	cert = flag.String("c", "", "cert file")
	key  = flag.String("k", "", "cert key file")
	host = flag.String("host", "", "host name")
	auth = flag.String("auth", "", "eg-> name:pass")
	max  = flag.Float64("max", 720, "max speed of connection (KB/s)")
)

func main() {
	flag.Parse()
	if *host != "" {
		*cert = fmt.Sprintf("/root/.caddy/acme/acme-v02.api.letsencrypt.org/sites/%s/%s.crt", *host, *host)
		*key = fmt.Sprintf("/root/.caddy/acme/acme-v02.api.letsencrypt.org/sites/%s/%s.key", *host, *host)
	}
	// max kb/s
	s := hox.NewServer(*addr, *auth, *cert, *key, *max)
	/*go func() {
		http.ListenAndServe(":6061", nil)
	}()*/
	s.Start()
}
