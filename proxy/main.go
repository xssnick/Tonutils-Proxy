package main

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/liteclient/adnl"
	"github.com/xssnick/tonutils-go/liteclient/adnl/dht"
	rldphttp "github.com/xssnick/tonutils-go/liteclient/adnl/rldp/http"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/dns"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

func appendHostToXForwardHeader(header http.Header, host string) {
	// If we aren't the first proxy retain prior
	// X-Forwarded-For information as a comma+space
	// separated list and fold multiple headers into one.
	if prior, ok := header["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + host
	}
	header.Set("X-Forwarded-For", host)
}

type proxy struct {
}

var client *http.Client

func (p *proxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		msg := "unsupported protocal scheme " + req.URL.Scheme
		http.Error(wr, msg, http.StatusBadRequest)
		return
	}

	//http: Request.RequestURI can't be set in client requests.
	//http://golang.org/src/pkg/net/http/client.go
	req.RequestURI = ""

	delHopHeaders(req.Header)

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		appendHostToXForwardHeader(req.Header, clientIP)
	}

	var c = http.DefaultClient
	if strings.HasSuffix(req.Host, ".ton") {
		log.Println("OVER RLDP", " ", req.Method, " ", req.URL)
		// proxy requests to ton using special client
		c = client
	} else {
		log.Println("OVER HTTP", " ", req.Method, " ", req.URL)
	}

	resp, err := c.Do(req)
	if err != nil {
		http.Error(wr, "RLDP Proxy Error:\n"+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	log.Println(req.RemoteAddr, " ", resp.Status)

	delHopHeaders(resp.Header)

	copyHeader(wr.Header(), resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
}

func startProxy(addr string, debug bool) {
	log.Println("Fetching TON network config...")
	cfg, err := liteclient.GetConfigFromUrl(context.Background(), "https://ton-blockchain.github.io/global.config.json")
	if err != nil {
		log.Println("Cannot fetch network config, error:", err.Error())
		return
	}

	if !debug {
		// omit internal logs
		adnl.Logger = func(v ...any) {}
	}

	var nodes []dht.NodeInfo
	for _, node := range cfg.DHT.StaticNodes.Nodes {
		ip := make(net.IP, 4)
		ii := int32(node.AddrList.Addrs[0].IP)
		binary.BigEndian.PutUint32(ip, uint32(ii))

		pp, err := base64.StdEncoding.DecodeString(node.ID.Key)
		if err != nil {
			continue
		}

		nodes = append(nodes, dht.NodeInfo{
			Address: ip.String() + ":" + fmt.Sprint(node.AddrList.Addrs[0].Port),
			Key:     pp,
		})
	}

	log.Println("Initialising DHT client...")
	dhtClient, err := dht.NewClient(10*time.Second, nodes)
	if err != nil {
		log.Println("Failed to init DHT client:", err.Error())
		return
	}

	log.Println("Initialising DNS resolver...")
	dnsClient, err := initDNSResolver(cfg)
	if err != nil {
		log.Println("Failed to init TON DNS resolver:", err.Error())
		return
	}

	log.Println("Initialising RLDP transport layer...")
	client = &http.Client{
		Transport: rldphttp.NewTransport(dhtClient, dnsClient),
	}

	log.Println("Starting proxy server on", addr)
	if err := http.ListenAndServe(addr, &proxy{}); err != nil {
		log.Fatal("Listen error:", err)
	}
}

func initDNSResolver(cfg *liteclient.GlobalConfig) (*dns.Client, error) {
	pool := liteclient.NewConnectionPool()

	// connect to testnet lite server
	err := pool.AddConnectionsFromConfig(context.Background(), cfg)
	if err != nil {
		return nil, err
	}

	// initialize ton api lite connection wrapper
	api := ton.NewAPIClient(pool)

	var root *address.Address
	for i := 0; i < 3; i++ { // retry to not get liteserver not found block err
		// get root dns address from network config
		root, err = dns.RootContractAddr(api)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		break
	}
	if err != nil {
		return nil, err
	}

	return dns.NewDNSClient(api, root), nil
}

func main() {
	var addr = flag.String("addr", "127.0.0.1:8080", "The addr of the proxy.")
	var debug = flag.Bool("debug", false, "Show additional logs")
	flag.Parse()

	startProxy(*addr, *debug)
}
