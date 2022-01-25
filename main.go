package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/koron/go-ssdp"
)

var ad *ssdp.Advertiser
var target, listen *string
var transcode *bool

func onSearch(m *ssdp.SearchMessage) {
	if strings.Contains(m.Type, "ssdp:all") || strings.Contains(m.Type, "service:ContentDirectory") || strings.Contains(m.Type, "service:ConnectionManager") || strings.Contains(m.Type, "device:MediaServer") {
		ad.Alive()
		log.Printf("Search: From=%s Type=%s\n", m.From.String(), m.Type)
	}
}

func rewriteBody(resp *http.Response) (err error) {
	for _, val := range resp.Header["Content-Type"] {
		if strings.Contains(val, "text/xml") {
			log.Println("Rewrote XML response")

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
		if strings.Contains(val, "audio/ogg") && *transcode {
			log.Println("OGG audio will be transcoded to FLAC")
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return err
			}
			err = resp.Body.Close()
			if err != nil {
				return err
			}

			cmd := exec.Command("ffmpeg", "-y", // Yes to all
				"-i", "pipe:0", // take stdin as input
				"-c:a", "flac", // use mp3 lame codec
				"-f", "flac", // using mp3 muxer (IMPORTANT, output data to pipe require manual muxer selecting)
				"-map_metadata", "0",
				"-sample_fmt", "s16",
				"pipe:1", // output to stdout
			)
			var stdout bytes.Buffer
			cmd.Stdout = &stdout        // stdout result will be written here
			stdin, _ := cmd.StdinPipe() // Open stdin pipe
			cmd.Start()                 // Start a process on another goroutine
			stdin.Write(b)              // pump audio data to stdin pipe
			stdin.Close()               // close the stdin, or ffmpeg will wait forever
			cmd.Wait()                  // wait until ffmpeg finish

			body := ioutil.NopCloser(bytes.NewReader(stdout.Bytes()))
			resp.Body = body
			resp.ContentLength = int64(len(stdout.Bytes()))
			resp.Header.Set("Content-Length", strconv.Itoa(len(stdout.Bytes())))
			resp.Header.Set("Content-Type", "audio/flac")
			return nil
		}

	}
	return nil
}

func main() {
	listen = flag.String("listen", "127.0.0.1:8080", "The IP to listen for requests")
	target = flag.String("target", "127.0.0.1:8201", "The IP and port of the target")
	transcode = flag.Bool("transcode", false, "Transcode unsupported audio (experimental)")
	flag.Parse()

	f, _ := os.Create("dlnaproxy." + strings.Replace(*target, ":", "_", -1) + ".pid")
	f.WriteString(fmt.Sprint(os.Getpid()))
	defer f.Close()

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
			log.Println(r.RemoteAddr + " requests " + r.URL.RequestURI())
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
