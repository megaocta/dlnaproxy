package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/koron/go-ssdp"
)

var ad *ssdp.Advertiser
var target, listen *string

func onSearch(m *ssdp.SearchMessage) {
	if strings.Contains(m.Type, "service:ContentDirectory") || strings.Contains(m.Type, "service:ConnectionManager") || strings.Contains(m.Type, "device:MediaServer") {
		ad.Alive()
		fmt.Printf("Search: From=%s Type=%s\n", m.From.String(), m.Type)
	}
}

func rewriteBody(resp *http.Response) (err error) {
	for _, val := range resp.Header["Content-Type"] {
		if strings.Contains(val, "text/xml") {
			fmt.Println("Rewrote XML response")

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			err = resp.Body.Close()
			if err != nil {
				return err
			}
			b = bytes.Replace(b, []byte("http://"+*target+"/"), []byte("http://"+*listen+"/"), -1) // replace original url with proxy url
			body := ioutil.NopCloser(bytes.NewReader(b))
			resp.Body = body
			resp.ContentLength = int64(len(b))
			resp.Header.Set("Content-Length", strconv.Itoa(len(b)))
			return nil
		}
	}
	return nil
}

func main() {
	listen = flag.String("listen", "127.0.0.1:8080", "The IP to listen for requests")
	target = flag.String("target", "127.0.0.1:8201", "The IP and port of the target")
	flag.Parse()

	f, _ := os.Create("dlnaproxy." + strings.Replace(*target, ":", "_", -1) + ".pid")
	f.WriteString(fmt.Sprint(os.Getpid()))

	var err error
	ad, err = ssdp.Advertise(
		"urn:schemas-upnp-org:service:ContentDirectory:1",                                         // send as "ST"
		fmt.Sprintf("uuid:%s::urn:schemas-upnp-org:service:ContentDirectory:1", uuid.NewString()), // send as "USN"
		fmt.Sprintf("http://%s/rootDesc.xml", *listen),                                            // send as "LOCATION"
		"Go DLNA proxy server", // send as "SERVER"
		1800)                   // send as "maxAge" in "CACHE-CONTROL"
	if err != nil {
		panic(err)
	}
	m := &ssdp.Monitor{
		Search: onSearch,
	}
	m.Start()

	remote, err := url.Parse("http://" + *target + "/")
	if err != nil {
		panic(err)
	}

	handler := func(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			fmt.Println(r.URL)
			r.Host = remote.Host
			p.ServeHTTP(w, r)
		}
	}

	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.ModifyResponse = rewriteBody
	http.HandleFunc("/", handler(proxy))
	err = http.ListenAndServe(*listen, nil)
	if err != nil {
		panic(err)
	}

}
