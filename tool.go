package hox

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

func parseReq(c *conn, free bool) (rawReqHeader bytes.Buffer, host, credential string, connect bool, err error) {
	req, err := http.ReadRequest(c.brc)
	if err != nil {
		err = &BadRequestError{"Read request timeout"}
		return
	}
	if req.Method == "CONNECT" {
		connect = true
		req.RequestURI = "https://" + req.RequestURI
	} else {
		req.Header.Del("Proxy-Connection")
	}
	// get remote host
	uriInfo, err := url.ParseRequestURI(req.RequestURI)
	if err != nil {
		return
	}
	credential = req.Header.Get("Proxy-Authorization")
	req.Header.Del("Proxy-Authorization")
	if uriInfo.Host == "" {
		host = req.Header.Get("Host")
	} else {
		if strings.Index(uriInfo.Host, ":") == -1 {
			host = uriInfo.Host + ":80"
		} else {
			host = uriInfo.Host
		}
	}
	req.Header.Add("HOST", host)
	// rebuild ReqHeader
	requestLine := fmt.Sprintf("%s %s %s\r\n", req.Method, req.URL.Path, req.Proto)
	rawReqHeader.WriteString(requestLine)
	req.Header.Add("Hox", "v1.1")
	for k, vs := range req.Header {
		for _, v := range vs {
			rawReqHeader.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
		}
	}
	rawReqHeader.WriteString("\r\n")
	return
}
