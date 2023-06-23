package proxy

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/adnl"
	"github.com/xssnick/tonutils-go/adnl/dht"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/dns"
	"github.com/xssnick/tonutils-proxy/proxy/transport"
	"github.com/xssnick/tonutils-storage/config"
	"github.com/xssnick/tonutils-storage/storage"
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
	if req.URL.Scheme == "" {
		// if no scheme - we check forwarded proto
		req.URL.Scheme = req.Header.Get("X-Forwarded-Proto")
	}

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
	if strings.HasSuffix(req.Host, ".ton") || strings.HasSuffix(req.Host, ".adnl") ||
		strings.HasSuffix(req.Host, ".t.me") || strings.HasSuffix(req.Host, ".bag") {
		log.Println("OVER RLDP", " ", req.Method, " ", req.URL)
		// proxy requests to ton using special client
		c = client
	} else {
		log.Println("OVER HTTP", " ", req.Method, " ", req.URL)
	}

	resp, err := c.Do(req)
	if err != nil {
		text := err.Error()
		if strings.Contains(text, "context deadline exceeded") {
			http.Error(wr, "TON Site "+req.URL.Host+" is not responding.", http.StatusBadGateway)
		} else {
			http.Error(wr, "RLDP Proxy Error:\n"+text, http.StatusBadGateway)
		}
		log.Println("cannot open", req.URL.String(), "| err:", text)
		return
	}
	defer resp.Body.Close()

	log.Println(req.RemoteAddr, " ", resp.Status)

	delHopHeaders(resp.Header)

	copyHeader(wr.Header(), resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
}

type State struct {
	Type    string
	State   string
	Stopped bool
}

func StartProxy(addr string, debug bool, res chan<- State) error {
	report := func(s State) {
		if res != nil {
			res <- s
		}
	}

	report(State{
		Type:  "loading",
		State: "Fetching network config...",
	})

	log.Println("Fetching TON network config...")
	lsCfg, err := liteclient.GetConfigFromUrl(context.Background(), "https://ton.org/global.config.json")
	if err != nil {
		log.Println("Failed to download ton config:", err.Error(), "; We will take it from static cache")
		lsCfg = &liteclient.GlobalConfig{}
		if err = json.NewDecoder(bytes.NewBufferString(config.FallbackNetworkConfig)).Decode(lsCfg); err != nil {
			return fmt.Errorf("failed to parse fallback ton config: %w", err)
		}
	}

	if !debug {
		// omit internal logs
		adnl.Logger = func(v ...any) {}
	}

	report(State{
		Type:  "loading",
		State: "Initializing DHT...",
	})

	log.Println("Initialising DHT client...")
	_, dhtAdnlKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return fmt.Errorf("failed to generate ed25519 dht adnl key: %w", err)
	}

	gateway := adnl.NewGateway(dhtAdnlKey)
	err = gateway.StartClient()
	if err != nil {
		return fmt.Errorf("failed to start adnl gateway: %w", err)
	}

	dhtClient, err := dht.NewClientFromConfig(gateway, lsCfg)
	if err != nil {
		return fmt.Errorf("failed to init DHT client: %w", err)
	}

	report(State{
		Type:  "loading",
		State: "Initializing DNS...",
	})

	log.Println("Initialising DNS resolver...")
	dnsClient, err := initDNSResolver(lsCfg)
	if err != nil {
		return fmt.Errorf("failed to init TON DNS resolver: %w", err)
	}

	report(State{
		Type:  "loading",
		State: "Initializing RLDP...",
	})

	log.Println("Initialising RLDP transport layer...")
	_, baseAdnlKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return fmt.Errorf("failed to generate ed25519 dht adnl key: %w", err)
	}

	gate := adnl.NewGateway(baseAdnlKey)
	if err = gate.StartClient(); err != nil {
		return fmt.Errorf("failed to init adnl gateway: %w", err)
	}

	// storage.Logger = log.Println
	srv := storage.NewServer(dhtClient, gate, baseAdnlKey, false, false)
	conn := storage.NewConnector(srv)

	store := transport.NewVirtualStorage()
	srv.SetStorage(store)

	client = &http.Client{
		Transport: transport.NewTransport(dhtClient, dnsClient, conn, store),
	}

	report(State{
		Type:  "ready",
		State: "Ready",
	})

	log.Println("Starting proxy server on", addr)

	server := http.Server{Addr: addr, Handler: &proxy{}}

	go func() {
		if err = server.ListenAndServe(); err != nil {
			log.Println("failed to init proxy server:", err.Error())

			text := "Failed, check logs"
			if strings.Contains(err.Error(), "address already in use") {
				text = "Port is already in use"
			}

			report(State{
				Type:    "error",
				State:   text,
				Stopped: true,
			})
		}
	}()

	return nil
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
